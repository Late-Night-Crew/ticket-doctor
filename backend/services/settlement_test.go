package services

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
	"ticketdoctor/pkg/footballdataorg"
)

// TestSettlementEndToEnd follows one matchday through the whole system
// (B4, FR-7.2): the fixtures import adds the games, a ticket is analysed, the
// results import later records the score on the same match row, and the
// track record counts the leg; a game not yet played stays pending, and a
// corrected score corrects the record.
func TestSettlementEndToEnd(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	dir := t.TempDir()
	copyTree(t, "testdata/import", dir)

	kickoff := testNow.Add(5 * time.Hour) // Friday 15:00 UTC
	fixtures := &fakeFixtures{fixtures: []footballdataorg.Fixture{
		apiFixture(1, "Chelsea FC", "Arsenal FC", "TIMED", kickoff),
		apiFixture(2, "Arsenal FC", "Everton FC", "TIMED", kickoff.Add(24*time.Hour)),
	}}
	importAll := func() {
		t.Helper()
		if _, err := newImporter(pool, dir).WithFixtures(fixtures).Run(ctx, withFixRange); err != nil {
			t.Fatalf("import: %v", err)
		}
	}
	matchID := func(home, away string) string {
		t.Helper()
		var id int64
		if err := pool.QueryRow(ctx, `SELECT m.id FROM matches m JOIN teams h ON h.id = m.home_team_id
			JOIN teams a ON a.id = m.away_team_id WHERE m.season = '2627' AND h.name = $1 AND a.name = $2`, home, away).Scan(&id); err != nil {
			t.Fatalf("%s v %s: %v", home, away, err)
		}
		return strconv.FormatInt(id, 10)
	}

	// 1. Fixtures arrive and a ticket is analysed before kick-off.
	importAll()
	chelseaArsenal, arsenalEverton := matchID("Chelsea", "Arsenal"), matchID("Arsenal", "Everton")
	tickets := NewTicketService(pool, engine.Default(), clock)
	if _, err := tickets.Analyse(ctx, AnalyseRequest{TicketID: ticket, Origin: "manual", Legs: []TicketLegInput{
		{MatchID: chelseaArsenal, Market: "HOME_WIN"},
		{MatchID: arsenalEverton, Market: "OVER_2_5"},
	}}); err != nil {
		t.Fatal(err)
	}
	results := NewResultsService(pool)
	if res, _ := results.Results(ctx, "", ""); len(res.Rows) != 0 {
		t.Fatalf("nothing played yet, but results = %+v", res.Rows)
	}

	// 2. After the game: football-data.co.uk lists the result (in UK time, so a
	// different kick-off than the fixture had), football-data.org says FINISHED.
	csv := filepath.Join(dir, "2627", "E0.csv")
	appendRow := func(row string) {
		t.Helper()
		data, err := os.ReadFile(csv)
		if err != nil {
			t.Fatal(err)
		}
		// The test file ends with blank rows; put the result before them.
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		at := len(lines)
		for at > 0 && strings.Trim(lines[at-1], ",") == "" {
			at--
		}
		lines = append(lines[:at], append([]string{row}, lines[at:]...)...)
		if err := os.WriteFile(csv, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	appendRow("E0,27/11/2026,15:00,Chelsea,Arsenal,2,1,H,2.6,3.4,2.7")
	fixtures.fixtures[0] = apiFixture(1, "Chelsea FC", "Arsenal FC", "FINISHED", kickoff)
	importAll()

	if again := matchID("Chelsea", "Arsenal"); again != chelseaArsenal {
		t.Fatalf("the result created match %s instead of updating %s", again, chelseaArsenal)
	}
	var status, source string
	if err := pool.QueryRow(ctx, `SELECT status, source FROM matches WHERE id = $1`, chelseaArsenal).Scan(&status, &source); err != nil ||
		status != "finished" || source != "football-data.co.uk" {
		t.Fatalf("Chelsea v Arsenal = %s from %s (%v), want finished from the results file", status, source, err)
	}

	res, err := results.Results(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 1 || res.Rows[0].Market != "HOME_WIN" || res.Rows[0].Original != (HitCount{Won: 1}) {
		t.Fatalf("after the matchday: %+v, want HOME_WIN won once and Arsenal v Everton still pending", res.Rows)
	}

	// 3. A corrected score (2-1 became 1-1) corrects the record on the next import.
	data, _ := os.ReadFile(csv)
	if err := os.WriteFile(csv, []byte(strings.Replace(string(data), "Chelsea,Arsenal,2,1,H", "Chelsea,Arsenal,1,1,D", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	importAll()
	res, _ = results.Results(ctx, "", "")
	if len(res.Rows) != 1 || res.Rows[0].Original != (HitCount{Lost: 1}) {
		t.Fatalf("after the correction: %+v, want HOME_WIN lost once", res.Rows)
	}
}
