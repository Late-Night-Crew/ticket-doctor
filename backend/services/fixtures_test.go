package services

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/pkg/footballdataorg"
)

// fakeFixtures returns the same fixtures for every league and records calls.
type fakeFixtures struct {
	fixtures []footballdataorg.Fixture
	err      error
	codes    []string
	from, to time.Time
}

func (f *fakeFixtures) Fixtures(_ context.Context, code string, from, to time.Time) ([]footballdataorg.Fixture, error) {
	f.codes = append(f.codes, code)
	f.from, f.to = from, to
	return f.fixtures, f.err
}

var (
	fixFrom      = time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	fixTo        = time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	withFixRange = ImportOptions{Leagues: both.Leagues, Seasons: both.Seasons, FixturesFrom: fixFrom, FixturesTo: fixTo}
	sat          = time.Date(2026, 11, 28, 15, 0, 0, 0, time.UTC)
)

// fixture builds an API fixture in 2026/27; names are football-data.org's
// official spellings, which differ from the results file's.
func fixture(id int64, home, away, apiStatus string, kickoff time.Time) footballdataorg.Fixture {
	status := map[string]footballdataorg.Status{
		"TIMED": footballdataorg.Scheduled, "SCHEDULED": footballdataorg.Scheduled,
		"POSTPONED": footballdataorg.Postponed, "FINISHED": footballdataorg.Skip,
	}[apiStatus]
	return footballdataorg.Fixture{
		ID: id, Kickoff: kickoff, Season: "2627", APIStatus: apiStatus, Status: status,
		Home: footballdataorg.Team{Name: home}, Away: footballdataorg.Team{Name: away},
	}
}

func baseFixtures() []footballdataorg.Fixture {
	return []footballdataorg.Fixture{
		fixture(1, "Chelsea FC", "Arsenal FC", "TIMED", sat),
		{ID: 2, Kickoff: sat.Add(2 * time.Hour), Season: "2627", APIStatus: "SCHEDULED", Status: footballdataorg.Scheduled,
			// The full name needs the alias table; the short name is a fallback.
			Home: footballdataorg.Team{Name: "Nottingham Forest FC", ShortName: "Nottingham"},
			Away: footballdataorg.Team{Name: "Everton FC", ShortName: "Everton"}},
		// Already played and imported as 1-2 from the results file.
		fixture(3, "Arsenal FC", "Chelsea FC", "TIMED", sat.Add(24*time.Hour)),
		fixture(4, "Everton FC", "Leeds United FC", "POSTPONED", sat),
		fixture(5, "Chelsea FC", "Everton FC", "FINISHED", sat),
	}
}

func importWithFixtures(t *testing.T, pool *pgxpool.Pool, src *fakeFixtures) (*ImportSummary, error) {
	t.Helper()
	return newImporter(pool, "testdata/import").WithFixtures(src).Run(context.Background(), withFixRange)
}

type matchRow struct {
	status  string
	kickoff time.Time
	goals   *int16
	source  string
}

func getMatch(t *testing.T, pool *pgxpool.Pool, home, away string) (matchRow, bool) {
	t.Helper()
	var m matchRow
	err := pool.QueryRow(context.Background(), `
		SELECT m.status, m.kickoff, m.home_goals, m.source FROM matches m
		JOIN teams h ON h.id = m.home_team_id JOIN teams a ON a.id = m.away_team_id
		WHERE m.season = '2627' AND h.name = $1 AND a.name = $2`, home, away).Scan(&m.status, &m.kickoff, &m.goals, &m.source)
	if err != nil {
		return m, false
	}
	return m, true
}

func TestFixturesImported(t *testing.T) {
	pool := dbtest.New(t)
	src := &fakeFixtures{fixtures: baseFixtures()}

	sum, err := importWithFixtures(t, pool, src)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(src.codes, []string{"PL"}) || !src.from.Equal(fixFrom) || !src.to.Equal(fixTo) {
		t.Fatalf("source called with %v %v %v", src.codes, src.from, src.to)
	}
	li := sum.Leagues[0]
	if li.Inserted != 18 {
		t.Fatalf("results inserted = %d, want 18", li.Inserted)
	}
	want := FixtureImport{Rows: 5, Inserted: 3, Unchanged: 1, Postponed: 1, Played: 1}
	if fx := li.Fixtures; fx == nil || fx.Rows != want.Rows || fx.Inserted != want.Inserted || fx.Updated != 0 ||
		fx.Unchanged != want.Unchanged || fx.Postponed != want.Postponed || fx.Played != want.Played || len(fx.Unmatched) != 0 {
		t.Fatalf("fixtures = %+v, want %+v", li.Fixtures, want)
	}
	if len(li.NewTeams) != 5 {
		t.Fatalf("new teams = %v: fixtures must not add clubs", li.NewTeams)
	}

	m, ok := getMatch(t, pool, "Chelsea", "Arsenal")
	if !ok || m.status != "scheduled" || !m.kickoff.Equal(sat) || m.goals != nil || m.source != "football-data.org" {
		t.Fatalf("Chelsea v Arsenal = %+v (found %v)", m, ok)
	}
	if m, _ := getMatch(t, pool, "Nott'm Forest", "Everton"); m.status != "scheduled" {
		t.Fatalf("Nott'm Forest v Everton = %+v, want scheduled", m)
	}
	if m, _ := getMatch(t, pool, "Everton", "Leeds"); m.status != "postponed" {
		t.Fatalf("Everton v Leeds = %+v, want postponed", m)
	}
	// The played game keeps its result: a fixture never overwrites it.
	if m, _ := getMatch(t, pool, "Arsenal", "Chelsea"); m.status != "finished" || m.goals == nil || *m.goals != 1 {
		t.Fatalf("Arsenal v Chelsea = %+v, want finished 1-2", m)
	}
	// A game the fixtures source reports as FINISHED is left to the results
	// import, which already saved it.
	if m, ok := getMatch(t, pool, "Chelsea", "Everton"); !ok || m.status != "finished" || m.source != "football-data.co.uk" {
		t.Fatalf("Chelsea v Everton = %+v (found %v), want the results row untouched", m, ok)
	}
}

func TestFixturesRerunAndChanges(t *testing.T) {
	pool := dbtest.New(t)
	if _, err := importWithFixtures(t, pool, &fakeFixtures{fixtures: baseFixtures()}); err != nil {
		t.Fatal(err)
	}

	// Same data: nothing changes.
	sum, err := importWithFixtures(t, pool, &fakeFixtures{fixtures: baseFixtures()})
	if err != nil {
		t.Fatal(err)
	}
	if fx := sum.Leagues[0].Fixtures; fx.Inserted != 0 || fx.Updated != 0 || fx.Unchanged != 4 {
		t.Fatalf("second run = %+v, want 4 unchanged", fx)
	}

	// Moved kick-off and a new date for the postponed game: both update.
	changed := baseFixtures()
	changed[0].Kickoff = sat.Add(-26 * time.Hour)
	changed[3] = fixture(4, "Everton FC", "Leeds United FC", "TIMED", sat.Add(96*time.Hour))
	sum, err = importWithFixtures(t, pool, &fakeFixtures{fixtures: changed})
	if err != nil {
		t.Fatal(err)
	}
	if fx := sum.Leagues[0].Fixtures; fx.Updated != 2 || fx.Unchanged != 2 || fx.Inserted != 0 {
		t.Fatalf("after changes = %+v, want 2 updated", fx)
	}
	if m, _ := getMatch(t, pool, "Chelsea", "Arsenal"); !m.kickoff.Equal(sat.Add(-26 * time.Hour)) {
		t.Fatalf("Chelsea v Arsenal kick-off = %v, want moved", m.kickoff)
	}
	if m, _ := getMatch(t, pool, "Everton", "Leeds"); m.status != "scheduled" || !m.kickoff.Equal(sat.Add(96*time.Hour)) {
		t.Fatalf("Everton v Leeds = %+v, want rescheduled", m)
	}
}

func TestFixturesUnmatchedTeamIsReportedNotCreated(t *testing.T) {
	pool := dbtest.New(t)
	fixtures := append(baseFixtures(),
		fixture(6, "Sunderland AFC", "Arsenal FC", "TIMED", sat),
		fixture(7, "Everton FC", "Sunderland AFC", "TIMED", sat))

	sum, err := importWithFixtures(t, pool, &fakeFixtures{fixtures: fixtures})
	if err == nil {
		t.Fatal("expected the import to fail")
	}
	if e := sum.Leagues[0].Error; !strings.Contains(e, "Sunderland AFC") || !strings.Contains(e, "alias") {
		t.Fatalf("league error = %q, want an unmatched-team error", e)
	}
	fx := sum.Leagues[0].Fixtures
	if !slices.Equal(fx.Unmatched, []string{"Sunderland AFC"}) || fx.Inserted != 3 {
		t.Fatalf("fixtures = %+v, want Sunderland unmatched and the other 3 saved", fx)
	}
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM teams WHERE name ILIKE 'sunderland%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("sunderland teams = %d (%v), want none created", n, err)
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM import_runs WHERE id = $1`, sum.RunID).Scan(&status); err != nil || status != "failed" {
		t.Fatalf("run status = %q (%v), want failed so the daily job flags it", status, err)
	}
}

func TestFixturesDownloadFailureKeepsResults(t *testing.T) {
	pool := dbtest.New(t)
	sum, err := importWithFixtures(t, pool, &fakeFixtures{err: errors.New("API key rejected")})
	if err == nil {
		t.Fatal("expected the import to fail")
	}
	li := sum.Leagues[0]
	if !strings.Contains(li.Error, "results were saved") {
		t.Fatalf("league error = %q", li.Error)
	}
	if li.Inserted != 18 || li.Profiles != 10 || li.Fixtures == nil || li.Fixtures.Error != "API key rejected" {
		t.Fatalf("league = %+v, fixtures %+v: results should still be saved", li, li.Fixtures)
	}
}

func TestFixturesNeedADateRange(t *testing.T) {
	pool := dbtest.New(t)
	imp := newImporter(pool, "testdata/import").WithFixtures(&fakeFixtures{})
	for name, opts := range map[string]ImportOptions{
		"no range":       both,
		"to before from": {Leagues: both.Leagues, Seasons: both.Seasons, FixturesFrom: fixTo, FixturesTo: fixFrom},
	} {
		if _, err := imp.Run(context.Background(), opts); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
