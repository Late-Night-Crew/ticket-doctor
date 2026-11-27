package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
)

func TestSavedTicketLifecycle(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ctx := context.Background()
	ac := w.match(t, "Arsenal", "Chelsea", testNow.Add(5*time.Hour))
	el := w.match(t, "Everton", "Leeds", testNow.Add(29*time.Hour))
	tickets := newTickets(w)
	now := testNow
	saved := NewSavedService(w.pool, func() time.Time { return now })

	// Nothing analysed yet: nothing to save.
	if _, err := saved.Save(ctx, ticket); !isStatus(err, 404) {
		t.Fatalf("save before analysing: %v, want 404", err)
	}

	// Analyse, accept a swap on Arsenal v Chelsea, analyse again, then save.
	first := analyse(t, tickets, TicketLegInput{MatchID: ac, Market: "HOME_WIN"}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	swap := first.Legs[0].Suggestions[0]
	if err := tickets.Accept(ctx, swap.ID); err != nil {
		t.Fatal(err)
	}
	analyse(t, tickets, TicketLegInput{MatchID: ac, Market: swap.Market}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})

	st, err := saved.Save(ctx, ticket)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.ID) != 22 || st.Status != Pending || st.ExpiresAt != nil || len(st.Legs) != 2 {
		t.Fatalf("saved = %+v", st)
	}
	leg := st.Legs[0]
	if leg.Fixture.ID != ac || leg.Original.Market != "HOME_WIN" || leg.Final.Market != swap.Market || leg.Original.Outcome != Pending {
		t.Fatalf("first leg = %+v, want user's HOME_WIN kept and final %s", leg, swap.Market)
	}
	var accepted bool
	for _, sg := range leg.Suggestions {
		accepted = accepted || (sg.Market == swap.Market && sg.Accepted)
	}
	if len(leg.Suggestions) == 0 || !accepted {
		t.Fatalf("suggestions shown = %+v, want the accepted swap among them", leg.Suggestions)
	}

	// Editing the ticket afterwards doesn't change the snapshot.
	analyse(t, tickets, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	if got, _ := saved.Get(ctx, st.ID); len(got.Legs) != 2 {
		t.Fatalf("snapshot changed by a later edit: %d legs", len(got.Legs))
	}

	// Saving again keeps the link and replaces the snapshot.
	again, err := saved.Save(ctx, ticket)
	if err != nil || again.ID != st.ID || len(again.Legs) != 1 {
		t.Fatalf("resave: %+v (%v)", again, err)
	}
	analyse(t, tickets, TicketLegInput{MatchID: ac, Market: swap.Market}, TicketLegInput{MatchID: el, Market: "OVER_1_5"})
	if again, err = saved.Save(ctx, ticket); err != nil || len(again.Legs) != 2 {
		t.Fatalf("resave with both games: %+v (%v)", again, err)
	}

	// Arsenal lose 0-2: the user's HOME_WIN lost, so the ticket as first
	// picked has lost; the other game is still to play.
	setScore(t, w, ac, 0, 2)
	got, err := saved.Get(ctx, st.ID)
	if err != nil {
		t.Fatal(err)
	}
	leg = got.Legs[0]
	if leg.Score == nil || *leg.Score != (Score{0, 2}) || leg.Original.Outcome != Lost || leg.Final.Outcome != settledAs(t, swap.Market, 0, 2) {
		t.Fatalf("after 0-2: %+v", leg)
	}
	sum := got.Summary
	if got.Status != Pending || sum.Settled != 1 || sum.OriginalTicket != Lost || sum.Original != (HitCount{Lost: 1}) {
		t.Fatalf("summary after one game: status %s %+v", got.Status, sum)
	}
	if pairs := sum.Pairs; pairs.SuggestionWonPickLost+pairs.BothLost != len(leg.Suggestions) || pairs.BothWon+pairs.PickWonSuggestionLost != 0 {
		t.Fatalf("pairs = %+v for %d suggestions on a lost pick", pairs, len(leg.Suggestions))
	}

	// The second game finishes 2-1: settled, kept 90 days from now.
	setScore(t, w, el, 2, 1)
	got, _ = saved.Get(ctx, st.ID)
	if got.Status != "settled" || got.ExpiresAt == nil || !got.ExpiresAt.Equal(now.AddDate(0, 0, 90)) || got.Summary.Settled != 2 {
		t.Fatalf("settled ticket: %s expires %v %+v", got.Status, got.ExpiresAt, got.Summary)
	}

	// The list leaves out unknown or malformed ids, and has no legs.
	list, err := saved.List(ctx, []string{"AAAAAAAAAAAAAAAAAAAAAA", st.ID, "not-an-id"})
	if err != nil || len(list) != 1 || list[0].ID != st.ID || list[0].Legs != nil {
		t.Fatalf("list = %+v (%v)", list, err)
	}

	// The daily clean-up marks it settled, then deletes it after 90 days.
	if n, d, err := saved.Cleanup(ctx); err != nil || n != 1 || d != 0 {
		t.Fatalf("clean-up today: settled %d deleted %d (%v)", n, d, err)
	}
	now = now.AddDate(0, 0, 91)
	if _, d, err := saved.Cleanup(ctx); err != nil || d != 1 {
		t.Fatalf("clean-up after 91 days: deleted %d (%v)", d, err)
	}
	if _, err := saved.Get(ctx, st.ID); !isStatus(err, 404) {
		t.Fatalf("expired ticket: %v", err)
	}
	// The ticket log, and so the public track record, is untouched.
	var legs int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM analysed_legs WHERE ticket_id = $1`, ticket).Scan(&legs); err != nil || legs != 2 {
		t.Fatalf("ticket log after deletion: %d rows (%v)", legs, err)
	}
}

func TestSavedTicketDeleteNeedsTheTicketID(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ctx := context.Background()
	ac := w.match(t, "Arsenal", "Chelsea", testNow.Add(5*time.Hour))
	analyse(t, newTickets(w), TicketLegInput{MatchID: ac, Market: "OVER_1_5"})
	saved := NewSavedService(w.pool, clock)
	st, err := saved.Save(ctx, ticket)
	if err != nil {
		t.Fatal(err)
	}
	// The link alone isn't enough to delete.
	if err := saved.Delete(ctx, st.ID, "someone-elses-ticket"); !isStatus(err, 404) {
		t.Fatalf("delete with another ticket id: %v", err)
	}
	if err := saved.Delete(ctx, st.ID, ticket); err != nil {
		t.Fatal(err)
	}
	if _, err := saved.Get(ctx, st.ID); !isStatus(err, 404) {
		t.Fatalf("after delete: %v", err)
	}
	if _, err := saved.List(ctx, make([]string, MaxSavedLookup+1)); !isStatus(err, 422) {
		t.Fatalf("too many ids: %v", err)
	}
}

func setScore(t *testing.T, w *world, matchID string, home, away int) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(),
		`UPDATE matches SET status = 'finished', home_goals = $2, away_goals = $3 WHERE id = $1`, matchID, home, away); err != nil {
		t.Fatal(err)
	}
}

func settledAs(t *testing.T, market string, home, away int) Outcome {
	t.Helper()
	o, err := engine.Settle(engine.Market(market), home, away)
	if err != nil {
		t.Fatal(err)
	}
	if o == engine.Won {
		return Won
	}
	return Lost
}

func isStatus(err error, status int) bool {
	var ue *UserError
	return errors.As(err, &ue) && ue.Status == status
}
