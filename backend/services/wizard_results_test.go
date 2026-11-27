package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
)

func TestWizardBuildsFromTheWindow(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	// testNow is Friday noon CAT. Two games today, one on Saturday, one already started.
	w.match(t, "Arsenal", "Chelsea", testNow.Add(3*time.Hour))
	w.match(t, "Everton", "Leeds", testNow.Add(6*time.Hour))
	w.match(t, "Chelsea", "Everton", testNow.Add(28*time.Hour))
	w.match(t, "Leeds", "Arsenal", testNow.Add(-time.Hour))
	s := NewWizardService(w.pool, engine.Default(), clock)
	ctx := context.Background()

	res, err := s.Build(ctx, WizardRequest{Path: "guided", Risk: "safe", Focus: "goals", Legs: 2, When: "today"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Legs) != 2 || res.ShortfallReason != nil {
		t.Fatalf("today: %+v", res)
	}
	for _, l := range res.Legs {
		if l.RiskBand != "low" || engine.Market(l.Market).Group() != engine.GroupGoals || l.FairOdds <= 1 || l.WindowN != 10 {
			t.Fatalf("safe goals leg = %+v", l)
		}
		if l.Fixture.Home.Name == "Chelsea" || l.Fixture.Home.Name == "Leeds" {
			t.Fatalf("leg outside today's window or already started: %+v", l.Fixture)
		}
	}

	// Asking for more than qualify gives a shorter ticket and says why.
	res, err = s.Build(ctx, WizardRequest{Path: "guided", Risk: "safe", Focus: "goals", Legs: 5, When: "weekend"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Legs) != 1 || res.ShortfallReason == nil || !strings.Contains(*res.ShortfallReason, "Only 1 of the 5 games") ||
		!strings.Contains(*res.ShortfallReason, "this weekend") {
		t.Fatalf("weekend shortfall: %+v %v", res.Legs, res.ShortfallReason)
	}

	// Beginners are capped at 4; out-of-range numbers are clamped, not rejected.
	res, err = s.Build(ctx, WizardRequest{Path: "beginner", Risk: "safe", Focus: "mix", Legs: 9, When: "today", Seed: 3})
	if err != nil || res.ShortfallReason == nil || !strings.Contains(*res.ShortfallReason, "of the 4 games") {
		t.Fatalf("beginner clamp: %+v %v", res, err)
	}

	for name, req := range map[string]WizardRequest{
		"bad path":  {Path: "expert", Risk: "safe", Focus: "mix", Legs: 2, When: "today"},
		"bad risk":  {Path: "guided", Risk: "yolo", Focus: "mix", Legs: 2, When: "today"},
		"bad focus": {Path: "guided", Risk: "safe", Focus: "cards", Legs: 2, When: "today"},
		"bad when":  {Path: "guided", Risk: "safe", Focus: "mix", Legs: 2, When: "upcoming"},
	} {
		if _, err := s.Build(ctx, req); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestResultsCountSettledPicksAndSuggestions(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ac := w.match(t, "Arsenal", "Chelsea", testNow.Add(time.Hour))
	el := w.match(t, "Everton", "Leeds", testNow.Add(time.Hour))
	ce := w.match(t, "Chelsea", "Everton", testNow.Add(time.Hour))
	tickets := newTickets(w)
	results := NewResultsService(w.pool)
	ctx := context.Background()

	// Analysed three times, with a swap accepted on Arsenal v Chelsea: the user's
	// pick (HOME_WIN) still counts once, as theirs.
	first := analyse(t, tickets,
		TicketLegInput{MatchID: ac, Market: "HOME_WIN"},
		TicketLegInput{MatchID: el, Market: "OVER_1_5"},
		TicketLegInput{MatchID: ce, Market: "DRAW"})
	swaps := first.Legs[0].Suggestions
	analyse(t, tickets,
		TicketLegInput{MatchID: ac, Market: swaps[0].Market},
		TicketLegInput{MatchID: el, Market: "OVER_1_5"},
		TicketLegInput{MatchID: ce, Market: "DRAW"})
	// Chelsea v Everton taken off before kick-off: not counted.
	analyse(t, tickets,
		TicketLegInput{MatchID: ac, Market: swaps[0].Market},
		TicketLegInput{MatchID: el, Market: "OVER_1_5"})

	// Nothing settled yet.
	res, err := results.Results(ctx, "", "")
	if err != nil || len(res.Rows) != 0 || res.UpdatedAt != nil {
		t.Fatalf("before any result: %+v %v", res, err)
	}

	w.finish(t, ac, 1, 2) // home win lost; goal markets (over 1.5, BTTS, ...) won
	w.finish(t, el, 0, 0) // over 1.5 lost
	w.finish(t, ce, 1, 1) // off the ticket: ignored
	w.set(t, `INSERT INTO import_runs (finished_at, status) VALUES (now(), 'succeeded')`)

	res, err = results.Results(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	byMarket := map[string]ResultsRow{}
	for _, r := range res.Rows {
		if r.LeagueID != "epl" {
			t.Fatalf("row in %s", r.LeagueID)
		}
		byMarket[r.Market] = r
	}
	if r := byMarket["HOME_WIN"]; r.Original != (HitCount{Lost: 1}) || r.OriginalWithSuggestion != (HitCount{Lost: 1}) {
		t.Fatalf("HOME_WIN = %+v, want the user's pick lost once, and it had a suggestion", r)
	}
	if r := byMarket["OVER_1_5"]; r.Original != (HitCount{Lost: 1}) || r.OriginalWithSuggestion != (HitCount{}) {
		t.Fatalf("OVER_1_5 = %+v, want lost once, no suggestion", r)
	}
	if _, ok := byMarket["DRAW"]; ok {
		t.Fatal("the removed Chelsea v Everton leg should not count")
	}
	var suggested HitCount
	for _, sw := range swaps {
		suggested.Won += byMarket[sw.Market].Suggested.Won
		suggested.Lost += byMarket[sw.Market].Suggested.Lost
		o, _ := engine.Settle(engine.Market(sw.Market), 1, 2)
		if o != engine.Won {
			t.Fatalf("test setup: expected %s to win at 1-2", sw.Market)
		}
	}
	if suggested != (HitCount{Won: len(swaps)}) {
		t.Fatalf("suggestions = %+v, want all %d won", suggested, len(swaps))
	}
	if res.UpdatedAt == nil {
		t.Fatal("updated_at should be the last import")
	}
	// Rows come in canonical market order.
	if res.Rows[0].Market != "HOME_WIN" {
		t.Fatalf("first row = %s, want HOME_WIN", res.Rows[0].Market)
	}

	// Filters.
	res, err = results.Results(ctx, "epl", "OVER_1_5")
	if err != nil || len(res.Rows) != 1 || res.Rows[0].Market != "OVER_1_5" {
		t.Fatalf("filtered: %+v %v", res, err)
	}
	if res, _ := results.Results(ctx, "laliga", ""); len(res.Rows) != 0 {
		t.Fatalf("laliga: %+v", res.Rows)
	}
	if _, err := results.Results(ctx, "seriea", ""); err == nil {
		t.Fatal("unknown league should be an error")
	}
	if _, err := results.Results(ctx, "", "DC_12"); err == nil {
		t.Fatal("unknown market should be an error")
	}
}

func TestFixturesByWindow(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	today := w.match(t, "Arsenal", "Chelsea", testNow.Add(3*time.Hour))
	saturday := w.match(t, "Everton", "Leeds", testNow.Add(28*time.Hour))
	w.match(t, "Chelsea", "Everton", testNow.Add(-time.Hour)) // started
	next := w.match(t, "Leeds", "Arsenal", testNow.Add(8*24*time.Hour))
	s := NewFixtureService(w.pool, clock)
	ctx := context.Background()

	ids := func(when Window) []string {
		t.Helper()
		fs, err := s.Fixtures(ctx, "epl", when)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, f := range fs {
			out = append(out, f.ID)
		}
		return out
	}
	if got := ids(Today); len(got) != 1 || got[0] != today {
		t.Errorf("today = %v", got)
	}
	if got := ids(Weekend); len(got) != 1 || got[0] != saturday {
		t.Errorf("weekend = %v", got)
	}
	if got := ids(Upcoming); len(got) != 3 || got[2] != next {
		t.Errorf("upcoming = %v", got)
	}
	if fs, _ := s.Fixtures(ctx, "laliga", Upcoming); len(fs) != 0 {
		t.Errorf("laliga = %v", fs)
	}
	if _, err := s.Fixtures(ctx, "seriea", Upcoming); err == nil {
		t.Error("unknown league should be a 404")
	}
	leagues, err := s.Leagues(ctx)
	if err != nil || len(leagues) != 3 {
		t.Fatalf("leagues = %v %v", leagues, err)
	}
}
