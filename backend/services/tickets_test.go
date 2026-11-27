package services

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
)

const ticket = "0b6f3c2e-6a1d-4b8e-9f1a-2c3d4e5f6a7b"

func newTickets(w *world) *TicketService { return NewTicketService(w.pool, engine.Default(), clock) }

func analyse(t *testing.T, s *TicketService, legs ...TicketLegInput) *AnalyseResponse {
	t.Helper()
	res, err := s.Analyse(context.Background(), AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: legs})
	if err != nil {
		t.Fatalf("analyse: %v", err)
	}
	return res
}

func odds(v float64) *float64 { return &v }

type logRow struct {
	original, final string
	onTicket        bool
	suggestions     int
	accepted        int
}

func ticketLog(t *testing.T, w *world, matchID string) logRow {
	t.Helper()
	var r logRow
	err := w.pool.QueryRow(context.Background(), `
		SELECT l.original_market, l.final_market, l.on_ticket,
		       (SELECT count(*) FROM suggestions s WHERE s.analysed_leg_id = l.id),
		       (SELECT count(*) FROM suggestions s WHERE s.analysed_leg_id = l.id AND s.accepted)
		FROM analysed_legs l WHERE l.ticket_id = $1 AND l.match_id = $2`, ticket, matchID).
		Scan(&r.original, &r.final, &r.onTicket, &r.suggestions, &r.accepted)
	if err != nil {
		t.Fatalf("ticket log for match %s: %v", matchID, err)
	}
	return r
}

func TestAnalyseRatesLegsAndSuggestsSwaps(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ac := w.match(t, "Arsenal", "Chelsea", testNow.Add(5*time.Hour))
	el := w.match(t, "Everton", "Leeds", testNow.Add(29*time.Hour))
	s := newTickets(w)

	res := analyse(t, s,
		TicketLegInput{MatchID: ac, Market: "HOME_WIN", Odds: odds(2.5)},
		TicketLegInput{MatchID: el, Market: "OVER_1_5"})

	if res.TicketID != ticket || len(res.Legs) != 2 {
		t.Fatalf("response = %+v", res)
	}
	win, over := res.Legs[0], res.Legs[1]
	want := estimateFor(t, engine.HomeWin)
	if math.Abs(win.Estimate-want.Estimate) > 1e-9 || win.RiskBand != string(want.Band) || win.Reason == "" {
		t.Fatalf("home win leg = %+v, want estimate %v band %s", win, want.Estimate, want.Band)
	}
	if win.Fixture.Home.Name != "Arsenal" || win.Fixture.Home.Form != "W-D-L-W-W" || win.MatchID != ac || win.WindowN != 10 || win.SampleSize != 10 {
		t.Fatalf("fixture details = %+v", win)
	}
	if win.ImpliedProb == nil || math.Abs(*win.ImpliedProb-0.4) > 1e-9 || win.IsValue == nil {
		t.Fatalf("odds maths missing: implied %v value %v", win.ImpliedProb, win.IsValue)
	}
	if over.ImpliedProb != nil || over.IsValue != nil {
		t.Fatal("a leg without odds has no implied probability or value")
	}
	if len(win.Suggestions) == 0 || win.Suggestions[0].ID == "" || win.Suggestions[0].Reason == "" {
		t.Fatalf("a High home win with goal-heavy teams should get swaps: %+v", win.Suggestions)
	}
	if len(over.Suggestions) != 0 {
		t.Fatalf("a Low leg gets no swaps: %+v", over.Suggestions)
	}
	wantP := win.Estimate * over.Estimate
	if math.Abs(res.Combined.Probability-wantP) > 1e-9 || res.Combined.OneIn != engine.OneIn(wantP) || res.Combined.Odds != nil {
		t.Fatalf("combined = %+v, want %v and no odds (one leg has none)", res.Combined, wantP)
	}
	if res.WeakestLegID == nil || *res.WeakestLegID != win.ID {
		t.Fatalf("weakest = %v, want the home win leg %s", res.WeakestLegID, win.ID)
	}
	if r := ticketLog(t, w, ac); r.original != "HOME_WIN" || r.final != "HOME_WIN" || !r.onTicket || r.suggestions != len(win.Suggestions) {
		t.Fatalf("log = %+v", r)
	}
}

func TestCombinedOddsWhenEveryLegHasThem(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	s := newTickets(w)
	res := analyse(t, s,
		TicketLegInput{MatchID: w.match(t, "Arsenal", "Chelsea", testNow.Add(time.Hour)), Market: "OVER_1_5", Odds: odds(1.25)},
		TicketLegInput{MatchID: w.match(t, "Everton", "Leeds", testNow.Add(time.Hour)), Market: "BTTS_YES", Odds: odds(1.804)})
	if res.Combined.Odds == nil || math.Abs(*res.Combined.Odds-1.25*1.8) > 1e-9 {
		t.Fatalf("combined odds = %v, want 2.25 (odds kept to 2 decimals)", res.Combined.Odds)
	}
}

func TestTicketLogCountsEachMatchOnce(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ac := w.match(t, "Arsenal", "Chelsea", testNow.Add(5*time.Hour))
	el := w.match(t, "Everton", "Leeds", testNow.Add(5*time.Hour))
	s := newTickets(w)
	first := analyse(t, s, TicketLegInput{MatchID: ac, Market: "HOME_WIN"}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})

	// Analysing again changes nothing in the log, and keeps ids.
	again := analyse(t, s, TicketLegInput{MatchID: ac, Market: "HOME_WIN"}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	if again.Legs[0].ID != first.Legs[0].ID || again.Legs[0].Suggestions[0].ID != first.Legs[0].Suggestions[0].ID {
		t.Fatal("re-analysis should keep leg and suggestion ids")
	}
	var legs int
	if err := w.pool.QueryRow(context.Background(), `SELECT count(*) FROM analysed_legs`).Scan(&legs); err != nil || legs != 2 {
		t.Fatalf("log rows = %d (%v), want 2", legs, err)
	}

	// Accepting a suggestion: the user's pick stays, the final pick changes.
	swap := first.Legs[0].Suggestions[0]
	if err := s.Accept(context.Background(), swap.ID); err != nil {
		t.Fatal(err)
	}
	analyse(t, s, TicketLegInput{MatchID: ac, Market: swap.Market}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	if r := ticketLog(t, w, ac); r.original != "HOME_WIN" || r.final != swap.Market || r.accepted != 1 {
		t.Fatalf("after accepting: %+v, want HOME_WIN kept, final %s, 1 accepted", r, swap.Market)
	}

	// Undo: back to the user's pick, the suggestion no longer counts as accepted.
	analyse(t, s, TicketLegInput{MatchID: ac, Market: "HOME_WIN"}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	if r := ticketLog(t, w, ac); r.original != "HOME_WIN" || r.final != "HOME_WIN" || r.accepted != 0 || r.suggestions == 0 {
		t.Fatalf("after undo: %+v", r)
	}

	// The user picks a different market themselves: it replaces their pick and
	// the old suggestions go.
	analyse(t, s, TicketLegInput{MatchID: ac, Market: "DRAW"}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	r := ticketLog(t, w, ac)
	if r.original != "DRAW" || r.final != "DRAW" {
		t.Fatalf("after the user's own change: %+v", r)
	}
	var stale int
	if err := w.pool.QueryRow(context.Background(), `SELECT count(*) FROM suggestions WHERE id = ANY ($1::bigint[])`,
		[]string{swap.ID, first.Legs[0].Suggestions[len(first.Legs[0].Suggestions)-1].ID}).Scan(&stale); err != nil || stale != 0 {
		t.Fatalf("suggestions made for the old pick = %d (%v), want gone", stale, err)
	}

	// Taking a match off the ticket stops it counting; putting it back restores it.
	analyse(t, s, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	if r := ticketLog(t, w, ac); r.onTicket {
		t.Fatal("removed leg should be off the ticket")
	}
	analyse(t, s, TicketLegInput{MatchID: ac, Market: "DRAW"}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	if r := ticketLog(t, w, ac); !r.onTicket {
		t.Fatal("re-added leg should be back on the ticket")
	}
}

func TestAnalyseRejectsBadTickets(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ok := w.match(t, "Arsenal", "Chelsea", testNow.Add(time.Hour))
	started := w.match(t, "Everton", "Leeds", testNow.Add(-time.Minute))
	postponed := w.match(t, "Chelsea", "Everton", testNow.Add(time.Hour))
	w.set(t, `UPDATE matches SET status = 'postponed' WHERE id = $1`, postponed)
	finished := w.match(t, "Leeds", "Arsenal", testNow.Add(-48*time.Hour))
	w.finish(t, finished, 1, 1)

	leg := func(id, market string) TicketLegInput { return TicketLegInput{MatchID: id, Market: market} }
	many := make([]TicketLegInput, 16)
	for i := range many {
		many[i] = leg(ok, "HOME_WIN")
	}
	s := newTickets(w)
	cases := []struct {
		name   string
		req    AnalyseRequest
		status int
		msg    string
	}{
		{"bad ticket id", AnalyseRequest{TicketID: "x", Origin: "manual", Legs: []TicketLegInput{leg(ok, "HOME_WIN")}}, 422, "valid id"},
		{"bad origin", AnalyseRequest{TicketID: ticket, Origin: "bot", Legs: []TicketLegInput{leg(ok, "HOME_WIN")}}, 422, "origin"},
		{"no legs", AnalyseRequest{TicketID: ticket, Origin: "manual"}, 422, "1–15"},
		{"16 legs", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: many}, 422, "1–15"},
		{"same match twice", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{leg(ok, "HOME_WIN"), leg(ok, "DRAW")}}, 422, "one leg per match"},
		{"unknown market", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{leg(ok, "DC_12")}}, 422, "Unknown market"},
		{"odds too low", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{{MatchID: ok, Market: "DRAW", Odds: odds(1)}}}, 422, "Odds"},
		{"not a match id", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{leg("abc", "DRAW")}}, 404, "Unknown match"},
		{"unknown match", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{leg("999999", "DRAW")}}, 404, "Unknown match"},
		{"kicked off", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{leg(started, "DRAW")}}, 422, "Everton v Leeds has already kicked off"},
		{"finished", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{leg(finished, "DRAW")}}, 422, "already kicked off"},
		{"postponed", AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{leg(postponed, "DRAW")}}, 422, "postponed"},
	}
	for _, c := range cases {
		_, err := s.Analyse(context.Background(), c.req)
		ue, isUser := AsUserError(err)
		if !isUser || ue.Status != c.status || !strings.Contains(ue.Msg, c.msg) {
			t.Errorf("%s: err = %v, want %d containing %q", c.name, err, c.status, c.msg)
		}
	}
	var n int
	if err := w.pool.QueryRow(context.Background(), `SELECT count(*) FROM analysed_legs`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rejected tickets logged %d legs", n)
	}
}

func TestTeamWithoutProfileIsLowConfidence(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	w.set(t, `DELETE FROM team_profiles WHERE team_id = $1`, w.teams["Chelsea"])
	res := analyse(t, newTickets(w), TicketLegInput{MatchID: w.match(t, "Arsenal", "Chelsea", testNow.Add(time.Hour)), Market: "OVER_1_5"})
	if l := res.Legs[0]; !l.LowConfidence || l.SampleSize != 0 {
		t.Fatalf("leg = %+v, want low confidence with sample 0", l)
	}
}

func TestAcceptUnknownSuggestion(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	for _, id := range []string{"abc", "0", "424242"} {
		err := newTickets(w).Accept(context.Background(), id)
		if ue, ok := AsUserError(err); !ok || ue.Status != 404 {
			t.Errorf("%q: err = %v, want 404", id, err)
		}
	}
}
