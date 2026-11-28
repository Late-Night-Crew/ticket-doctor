package services

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/pkg/footballdataorg"
)

func scored(f footballdataorg.Fixture, home, away int) footballdataorg.Fixture {
	f.Score = &footballdataorg.Score{Home: home, Away: away}
	return f
}

func TestFullImportFillsInScoresTheResultsFileLacks(t *testing.T) {
	pool := dbtest.New(t)
	// The fixtures are known before kick-off, from an earlier import.
	if _, err := importWithFixtures(t, pool, &fakeFixtures{fixtures: baseFixtures()}); err != nil {
		t.Fatal(err)
	}
	fixtures := baseFixtures()
	// Played, and not in the results file yet: football-data.org's score is used.
	fixtures[0] = scored(apiFixture(1, "Chelsea FC", "Arsenal FC", "FINISHED", sat), 3, 0)
	// In the results file as 1-2: the file wins, whatever the API says.
	fixtures[2] = scored(apiFixture(3, "Arsenal FC", "Chelsea FC", "FINISHED", sat), 5, 5)

	sum, err := importWithFixtures(t, pool, &fakeFixtures{fixtures: fixtures})
	if err != nil {
		t.Fatal(err)
	}
	if fx := sum.Leagues[0].Fixtures; fx.Results != 1 || fx.Played != 3 {
		t.Fatalf("fixtures = %+v, want 1 result from 3 played", fx)
	}
	if m, _ := getMatch(t, pool, "Chelsea", "Arsenal"); m.status != "finished" || *m.goals != 3 || m.source != "football-data.org" {
		t.Fatalf("Chelsea v Arsenal = %+v, want finished 3-0 from football-data.org", m)
	}
	if m, _ := getMatch(t, pool, "Arsenal", "Chelsea"); *m.goals != 1 || m.source != "football-data.co.uk" {
		t.Fatalf("Arsenal v Chelsea = %+v, want the results file's 1-2", m)
	}
}

func TestResultsFileCorrectsAQuickScore(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if _, err := newImporter(pool, "testdata/import").Run(ctx, both); err != nil {
		t.Fatal(err)
	}
	// Pretend the quick check saved a wrong score before the file arrived.
	if _, err := pool.Exec(ctx, `
		UPDATE matches m SET home_goals = 2, away_goals = 2, source = 'football-data.org'
		FROM teams h, teams a
		WHERE h.id = m.home_team_id AND a.id = m.away_team_id AND m.season = '2627'
		  AND h.name = 'Arsenal' AND a.name = 'Chelsea'`); err != nil {
		t.Fatal(err)
	}

	sum, err := newImporter(pool, "testdata/import").Run(ctx, both)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Arsenal v Chelsea 2-2 → 1-2 (was football-data.org)"}
	if got := sum.Leagues[0].Corrected; !slices.Equal(got, want) {
		t.Fatalf("corrected = %q, want %q", got, want)
	}
	if m, _ := getMatch(t, pool, "Arsenal", "Chelsea"); *m.goals != 1 || m.source != "football-data.co.uk" {
		t.Fatalf("Arsenal v Chelsea = %+v, want the file's 1-2", m)
	}
	// Agreeing results aren't reported.
	if sum, _ := newImporter(pool, "testdata/import").Run(ctx, both); len(sum.Leagues[0].Corrected) != 0 {
		t.Fatalf("third run corrected %q, want nothing", sum.Leagues[0].Corrected)
	}
}

func TestQuickScores(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if _, err := importWithFixtures(t, pool, &fakeFixtures{fixtures: baseFixtures()}); err != nil {
		t.Fatal(err)
	}

	// Three hours after the 15:00 games: Chelsea v Arsenal is over; Forest v
	// Everton (17:00) kicked off too recently to be asked about.
	now := sat.Add(3 * time.Hour)
	src := &fakeFixtures{fixtures: []footballdataorg.Fixture{
		scored(apiFixture(1, "Chelsea FC", "Arsenal FC", "FINISHED", sat), 2, 0),
		{ID: 2, Kickoff: sat.Add(2 * time.Hour), Season: "2627", APIStatus: "IN_PLAY", Status: footballdataorg.Skip,
			Home: footballdataorg.Team{Name: "Nottingham Forest FC"}, Away: footballdataorg.Team{Name: "Everton FC"}},
	}}
	homeSample := func() (n int16) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT p.sample_size FROM team_profiles p JOIN teams t ON t.id = p.team_id
			WHERE t.name = 'Chelsea' AND p.venue = 'home'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := homeSample()
	imp := newImporter(pool, "testdata/import").WithFixtures(src)
	sum, err := imp.QuickScores(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	day := sat.Truncate(24 * time.Hour)
	if !slices.Equal(src.codes, []string{"PL"}) || !src.from.Equal(day) || !src.to.Equal(day) {
		t.Fatalf("asked for %v from %v to %v, want PL on the day", src.codes, src.from, src.to)
	}
	if sum.RunID == 0 || sum.Results() != 1 || sum.Leagues[0].Profiles != 10 {
		t.Fatalf("summary = %+v, want 1 result and profiles rebuilt", sum)
	}
	if m, _ := getMatch(t, pool, "Chelsea", "Arsenal"); m.status != "finished" || *m.goals != 2 || m.source != "football-data.org" {
		t.Fatalf("Chelsea v Arsenal = %+v, want finished 2-0", m)
	}
	if m, _ := getMatch(t, pool, "Nott'm Forest", "Everton"); m.status != "scheduled" {
		t.Fatalf("Forest v Everton = %+v, want still scheduled while in play", m)
	}
	var kind, status string
	if err := pool.QueryRow(ctx, `SELECT kind, status FROM import_runs WHERE id = $1`, sum.RunID).Scan(&kind, &status); err != nil ||
		kind != "scores" || status != "succeeded" {
		t.Fatalf("run = %s %s (%v), want a succeeded scores run", kind, status, err)
	}
	// The team profiles now include the new result.
	if after := homeSample(); after != before+1 {
		t.Fatalf("Chelsea home sample = %d, want %d with the new game", after, before+1)
	}

	// Saved tickets show when results were last checked.
	checked, err := NewSavedService(pool, time.Now).q.LastResultsCheck(ctx)
	if err != nil || checked == nil {
		t.Fatalf("last results check = %v (%v)", checked, err)
	}

	// Days later nothing is awaiting a result (the postponed game is outside
	// the window): no request, nothing recorded.
	src.codes = nil
	sum, err = imp.QuickScores(ctx, now.Add(5*24*time.Hour))
	if err != nil || sum.RunID != 0 || len(src.codes) != 0 {
		t.Fatalf("idle check = %+v (%v), asked %v; want no request", sum, err, src.codes)
	}
}

func TestQuickScoresReportsUnmatchedClubs(t *testing.T) {
	pool := dbtest.New(t)
	ctx := context.Background()
	if _, err := importWithFixtures(t, pool, &fakeFixtures{fixtures: baseFixtures()}); err != nil {
		t.Fatal(err)
	}
	imp := newImporter(pool, "testdata/import").WithFixtures(&fakeFixtures{fixtures: []footballdataorg.Fixture{
		scored(apiFixture(1, "Chelsea FC", "Arsenal FC", "FINISHED", sat), 1, 1),
		scored(apiFixture(9, "Sunderland AFC", "Arsenal FC", "FINISHED", sat), 0, 4),
	}})
	sum, err := imp.QuickScores(ctx, sat.Add(3*time.Hour))
	if err == nil || !strings.Contains(sum.Leagues[0].Error, "Sunderland AFC") {
		t.Fatalf("err = %v, summary %+v: want the unmatched club reported", err, sum)
	}
	if sum.Leagues[0].Results != 1 {
		t.Fatalf("results = %d, want the matched game still saved", sum.Leagues[0].Results)
	}
}

func TestQuickScoresNeedASource(t *testing.T) {
	if _, err := newImporter(dbtest.New(t), "testdata/import").QuickScores(context.Background(), sat); err == nil {
		t.Fatal("expected an error without a fixtures source")
	}
}
