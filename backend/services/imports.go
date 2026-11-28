// Package services holds business logic behind interfaces, one file per area.
// Services read and write through dbsqlc and use the engine package for rules.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
	"ticketdoctor/logging/loggedmodule"
	"ticketdoctor/pkg/footballdata"
	"ticketdoctor/pkg/footballdataorg"
	"ticketdoctor/pkg/teamnames"
)

// ResultsSource supplies one league season of results.
type ResultsSource interface {
	Results(ctx context.Context, division, season string) ([]footballdata.Match, []footballdata.RowError, error)
}

// DirSource reads results from <Dir>/<season>/<division>.csv instead of the
// internet: for tests, and for importing files downloaded by hand.
type DirSource struct{ Dir string }

func (d DirSource) Results(_ context.Context, division, season string) ([]footballdata.Match, []footballdata.RowError, error) {
	f, err := os.Open(filepath.Join(d.Dir, season, division+".csv"))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	return footballdata.Parse(f)
}

// FixturesSource supplies a league's upcoming fixtures between two dates.
type FixturesSource interface {
	Fixtures(ctx context.Context, code string, from, to time.Time) ([]footballdataorg.Fixture, error)
}

// ImportOptions chooses what to import. Empty Leagues means every league with
// a football-data.co.uk code.
type ImportOptions struct {
	Leagues []string
	Seasons []string
	// FixturesFrom and FixturesTo are the kick-off dates to import fixtures
	// for. Used only when the service has a fixtures source.
	FixturesFrom, FixturesTo time.Time
}

// LeagueImport is what an import did for one league.
type LeagueImport struct {
	League        string   `json:"league"`
	Seasons       []string `json:"seasons"`
	Rows          int      `json:"rows"`
	Inserted      int      `json:"inserted"`
	Updated       int      `json:"updated"`
	Unchanged     int      `json:"unchanged"`
	SkippedRows   int      `json:"skipped_rows"`
	SkippedSample []string `json:"skipped_sample,omitempty"`
	NewTeams      []string `json:"new_teams,omitempty"`
	// Corrected lists games where the results file changed a score that came
	// from football-data.org or an admin, e.g. "Arsenal v Chelsea 2-1 → 2-2
	// (was football-data.org)". The file wins (addendum A10).
	Corrected []string `json:"corrected,omitempty"`
	Profiles  int      `json:"profiles"`
	// LowConfidence lists teams with a home or away profile below the cut-off.
	LowConfidence []string `json:"low_confidence,omitempty"`
	// Fixtures is nil when fixtures weren't imported.
	Fixtures *FixtureImport `json:"fixtures,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// FixtureImport is what the fixtures part of a league import did.
type FixtureImport struct {
	Rows      int `json:"rows"`
	Inserted  int `json:"inserted"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	// Postponed counts fixtures the source marks postponed, suspended or cancelled.
	Postponed int `json:"postponed"`
	// Played counts games in play or finished, left to the results import.
	Played int `json:"played"`
	// Results counts finished games given football-data.org's score because
	// the results file doesn't have them yet (addendum A10).
	Results int `json:"results"`
	// Unmatched lists club names that matched no team. Those fixtures are
	// skipped; fix them with an alias in pkg/teamnames.
	Unmatched []string `json:"unmatched,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// ImportSummary is stored on the import_runs row and logged.
type ImportSummary struct {
	RunID   int64          `json:"run_id"`
	Leagues []LeagueImport `json:"leagues"`
}

// ImportService loads results into the database and rebuilds team profiles
// (FR-1.1, FR-1.3, FR-1.5, FR-2).
type ImportService struct {
	pool     *pgxpool.Pool
	q        *dbsqlc.Queries
	source   ResultsSource
	fixtures FixturesSource // nil: results only
	engine   *engine.Engine
	logger   *slog.Logger
}

func NewImportService(pool *pgxpool.Pool, source ResultsSource, eng *engine.Engine, logger *slog.Logger) *ImportService {
	return &ImportService{pool: pool, q: dbsqlc.New(pool), source: source, engine: eng, logger: logger}
}

// WithFixtures makes the import also load upcoming fixtures (FR-1.2) for
// leagues with a football-data.org code.
func (s *ImportService) WithFixtures(src FixturesSource) *ImportService {
	s.fixtures = src
	return s
}

// maxSkippedSample caps how many skipped-row messages are kept per league.
const maxSkippedSample = 10

// Run imports each league in its own transaction: a failure in one league is
// recorded and the others still import. The run is recorded in import_runs and
// fails if any league failed.
func (s *ImportService) Run(ctx context.Context, opts ImportOptions) (*ImportSummary, error) {
	if len(opts.Seasons) == 0 {
		return nil, errors.New("no seasons to import")
	}
	for _, season := range opts.Seasons {
		if err := footballdata.ValidSeason(season); err != nil {
			return nil, err
		}
	}
	if s.fixtures != nil && (opts.FixturesFrom.IsZero() || opts.FixturesTo.Before(opts.FixturesFrom)) {
		return nil, errors.New("fixtures need a date range (FixturesFrom up to FixturesTo)")
	}
	leagues, err := s.leagues(ctx, opts.Leagues)
	if err != nil {
		return nil, err
	}

	runID, err := s.q.StartImportRun(ctx, "import")
	if err != nil {
		return nil, fmt.Errorf("record import run: %w", err)
	}
	summary := &ImportSummary{RunID: runID}

	var failed []string
	for _, l := range leagues {
		li, err := s.importLeague(ctx, l, opts)
		if err != nil {
			li.Error = err.Error()
			failed = append(failed, l.ID)
			s.logger.Error("league import failed", loggedmodule.Attr(loggedmodule.Import), "league", l.ID, "error", err)
		}
		summary.Leagues = append(summary.Leagues, li)
	}

	status, errText := "succeeded", (*string)(nil)
	var runErr error
	if len(failed) > 0 {
		status = "failed"
		runErr = fmt.Errorf("import failed for %v", failed)
		msg := runErr.Error()
		errText = &msg
	}
	body, _ := json.Marshal(summary)
	// Record the outcome even if the caller's context was cancelled mid-run.
	if err := s.q.FinishImportRun(context.WithoutCancel(ctx), dbsqlc.FinishImportRunParams{
		ID: runID, Status: status, Summary: body, Error: errText,
	}); err != nil {
		return summary, errors.Join(runErr, fmt.Errorf("record import result: %w", err))
	}
	return summary, runErr
}

func (s *ImportService) leagues(ctx context.Context, only []string) ([]dbsqlc.League, error) {
	all, err := s.q.ListLeagues(ctx)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, id := range only {
		want[id] = true
	}
	var out []dbsqlc.League
	for _, l := range all {
		if l.FdCode == nil || (len(only) > 0 && !want[l.ID]) {
			continue
		}
		out = append(out, l)
		delete(want, l.ID)
	}
	if len(want) > 0 {
		var unknown []string
		for id := range want {
			unknown = append(unknown, id)
		}
		sort.Strings(unknown)
		return nil, fmt.Errorf("unknown or non-CSV league(s): %v", unknown)
	}
	return out, nil
}

func (s *ImportService) importLeague(ctx context.Context, l dbsqlc.League, opts ImportOptions) (LeagueImport, error) {
	seasons := opts.Seasons
	li := LeagueImport{League: l.ID, Seasons: seasons}

	// Download everything first, so no transaction stays open during network calls.
	type seasonRows struct {
		season  string
		matches []footballdata.Match
	}
	var data []seasonRows
	for _, season := range seasons {
		matches, skipped, err := s.source.Results(ctx, *l.FdCode, season)
		if err != nil {
			return li, fmt.Errorf("season %s: %w", season, err)
		}
		for _, sk := range skipped {
			li.SkippedRows++
			if len(li.SkippedSample) < maxSkippedSample {
				li.SkippedSample = append(li.SkippedSample, fmt.Sprintf("%s %s", season, sk.Error()))
			}
		}
		li.Rows += len(matches)
		data = append(data, seasonRows{season, matches})
	}

	// A fixtures download failure doesn't stop the results: they're saved, and
	// the league is still reported as failed so the daily job shows the problem.
	var fixtures []footballdataorg.Fixture
	var fixturesErr error
	if s.fixtures != nil && l.FdoCode != nil {
		li.Fixtures = &FixtureImport{}
		fixtures, fixturesErr = s.fixtures.Fixtures(ctx, *l.FdoCode, opts.FixturesFrom, opts.FixturesTo)
		if fixturesErr != nil {
			li.Fixtures.Error = fixturesErr.Error()
		}
		li.Fixtures.Rows = len(fixtures)
	}

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		teams, err := newTeamResolver(ctx, q, l.ID)
		if err != nil {
			return err
		}
		provisional, err := provisionalResults(ctx, q, l.ID)
		if err != nil {
			return err
		}
		for _, sd := range data {
			for _, m := range sd.matches {
				if m.Division != *l.FdCode {
					return fmt.Errorf("season %s line %d: division %q in a %s file", sd.season, m.Line, m.Division, *l.FdCode)
				}
				home, err := teams.resolve(ctx, m.HomeTeam)
				if err != nil {
					return err
				}
				away, err := teams.resolve(ctx, m.AwayTeam)
				if err != nil {
					return err
				}
				hg, ag := int16(m.HomeGoals), int16(m.AwayGoals)
				if p, ok := provisional[matchKey{sd.season, home, away}]; ok && (p.home != hg || p.away != ag) {
					li.Corrected = append(li.Corrected, fmt.Sprintf("%s v %s %d-%d → %d-%d (was %s)",
						m.HomeTeam, m.AwayTeam, p.home, p.away, hg, ag, p.source))
				}
				row, err := q.UpsertMatch(ctx, dbsqlc.UpsertMatchParams{
					LeagueID: l.ID, Season: sd.season, Kickoff: m.Kickoff,
					HomeTeamID: home, AwayTeamID: away, Status: "finished",
					HomeGoals: &hg, AwayGoals: &ag,
					OddsHome: m.OddsHome, OddsDraw: m.OddsDraw, OddsAway: m.OddsAway,
					Source: "football-data.co.uk",
				})
				switch {
				case errors.Is(err, pgx.ErrNoRows):
					li.Unchanged++
				case err != nil:
					return fmt.Errorf("save %s v %s (%s line %d): %w", m.HomeTeam, m.AwayTeam, sd.season, m.Line, err)
				case row.Inserted:
					li.Inserted++
				default:
					li.Updated++
				}
			}
		}
		li.NewTeams = teams.created

		// After the results, so a game that has just finished is already saved
		// as finished and the fixture can't overwrite it.
		if err := s.saveFixtures(ctx, q, l.ID, teams, fixtures, li.Fixtures); err != nil {
			return err
		}

		profiles, low, err := s.rebuildProfiles(ctx, q, l.ID)
		li.Profiles, li.LowConfidence = profiles, low
		return err
	})
	if err != nil {
		// The transaction rolled back: nothing from this league was saved.
		li.Inserted, li.Updated, li.Unchanged, li.NewTeams, li.Profiles, li.LowConfidence, li.Corrected = 0, 0, 0, nil, 0, nil, nil
		if li.Fixtures != nil {
			li.Fixtures = &FixtureImport{Rows: li.Fixtures.Rows, Error: li.Fixtures.Error}
		}
		return li, err
	}
	s.logger.Info("league imported", loggedmodule.Attr(loggedmodule.Import),
		"league", l.ID, "rows", li.Rows, "inserted", li.Inserted, "updated", li.Updated,
		"unchanged", li.Unchanged, "skipped_rows", li.SkippedRows, "new_teams", len(li.NewTeams),
		"profiles", li.Profiles, "low_confidence_teams", len(li.LowConfidence), "corrected", li.Corrected)
	if fx := li.Fixtures; fx != nil {
		s.logger.Info("fixtures imported", loggedmodule.Attr(loggedmodule.Import),
			"league", l.ID, "rows", fx.Rows, "inserted", fx.Inserted, "updated", fx.Updated,
			"unchanged", fx.Unchanged, "postponed", fx.Postponed, "played", fx.Played, "results", fx.Results, "unmatched", fx.Unmatched)
	}
	switch {
	case fixturesErr != nil:
		return li, fmt.Errorf("fixtures (results were saved): %w", fixturesErr)
	case li.Fixtures != nil && len(li.Fixtures.Unmatched) > 0:
		return li, fmt.Errorf("fixtures: no team matches %q; add an alias in pkg/teamnames (the other fixtures were saved)", li.Fixtures.Unmatched)
	}
	return li, nil
}

// saveFixtures upserts upcoming and postponed fixtures. It never adds teams:
// every club in a league has results, so a name that matches nothing is a
// spelling the alias table doesn't know yet, and adding it would create a
// duplicate club.
func (s *ImportService) saveFixtures(ctx context.Context, q *dbsqlc.Queries, league string, teams *teamResolver, fixtures []footballdataorg.Fixture, fx *FixtureImport) error {
	if fx == nil { // results-only import
		return nil
	}
	unmatched := map[string]bool{}
	for _, f := range fixtures {
		if f.Status == footballdataorg.Skip {
			fx.Played++
			// A finished game the results file hasn't got yet: use this score
			// until it arrives. Runs after the results, so it never replaces one.
			saved, err := saveScore(ctx, q, league, teams, f)
			if err != nil {
				return err
			}
			if saved {
				fx.Results++
			}
			continue
		}
		home, okHome := teams.find(f.Home.Name, f.Home.ShortName)
		away, okAway := teams.find(f.Away.Name, f.Away.ShortName)
		if !okHome {
			unmatched[f.Home.Name] = true
		}
		if !okAway {
			unmatched[f.Away.Name] = true
		}
		if !okHome || !okAway {
			continue
		}
		if f.Status == footballdataorg.Postponed {
			fx.Postponed++
		}
		row, err := q.UpsertFixture(ctx, dbsqlc.UpsertFixtureParams{
			LeagueID: league, Season: f.Season, Kickoff: f.Kickoff,
			HomeTeamID: home, AwayTeamID: away, Status: string(f.Status),
			Source: "football-data.org",
		})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			fx.Unchanged++
		case err != nil:
			return fmt.Errorf("save fixture %s v %s (%d): %w", f.Home.Name, f.Away.Name, f.ID, err)
		case row.Inserted:
			fx.Inserted++
		default:
			fx.Updated++
		}
	}
	for name := range unmatched {
		fx.Unmatched = append(fx.Unmatched, name)
	}
	sort.Strings(fx.Unmatched)
	return nil
}

// saveScore stores a finished fixture's full-time score if the game has no
// result yet. It reports false, without an error, for a fixture with no score
// or with a club it can't match (the fixtures import reports those).
func saveScore(ctx context.Context, q *dbsqlc.Queries, league string, teams *teamResolver, f footballdataorg.Fixture) (bool, error) {
	if f.Score == nil {
		return false, nil
	}
	home, okHome := teams.find(f.Home.Name, f.Home.ShortName)
	away, okAway := teams.find(f.Away.Name, f.Away.ShortName)
	if !okHome || !okAway {
		return false, nil
	}
	hg, ag := int16(f.Score.Home), int16(f.Score.Away)
	_, err := q.SetQuickResult(ctx, dbsqlc.SetQuickResultParams{
		HomeGoals: &hg, AwayGoals: &ag, Kickoff: f.Kickoff,
		LeagueID: league, Season: f.Season, HomeTeamID: home, AwayTeamID: away,
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows): // already has a result, or not in our data
		return false, nil
	case err != nil:
		return false, fmt.Errorf("save score %s v %s (%d): %w", f.Home.Name, f.Away.Name, f.ID, err)
	}
	return true, nil
}

type matchKey struct {
	season     string
	home, away int64
}

type provisionalResult struct {
	home, away int16
	source     string
}

// provisionalResults are a league's finished games whose score didn't come
// from the results file.
func provisionalResults(ctx context.Context, q *dbsqlc.Queries, league string) (map[matchKey]provisionalResult, error) {
	rows, err := q.ListProvisionalResults(ctx, league)
	if err != nil {
		return nil, err
	}
	out := make(map[matchKey]provisionalResult, len(rows))
	for _, r := range rows {
		out[matchKey{r.Season, r.HomeTeamID, r.AwayTeamID}] = provisionalResult{*r.HomeGoals, *r.AwayGoals, r.Source}
	}
	return out, nil
}

// rebuildProfiles recomputes every team's home and away profile and last-5
// form from the league's finished matches (FR-2).
func (s *ImportService) rebuildProfiles(ctx context.Context, q *dbsqlc.Queries, league string) (int, []string, error) {
	teams, err := q.ListTeamsByLeague(ctx, league)
	if err != nil {
		return 0, nil, err
	}
	rows, err := q.ListFinishedMatchesByLeague(ctx, league)
	if err != nil {
		return 0, nil, err
	}
	home := map[int64][]engine.Result{}
	away := map[int64][]engine.Result{}
	all := map[int64][]engine.Result{}
	var leagueHome, leagueAway []engine.Result
	for _, r := range rows {
		hg, ag := int(*r.HomeGoals), int(*r.AwayGoals)
		h := engine.Result{Date: r.Kickoff, GoalsFor: hg, GoalsAgainst: ag}
		a := engine.Result{Date: r.Kickoff, GoalsFor: ag, GoalsAgainst: hg}
		home[r.HomeTeamID] = append(home[r.HomeTeamID], h)
		away[r.AwayTeamID] = append(away[r.AwayTeamID], a)
		all[r.HomeTeamID] = append(all[r.HomeTeamID], h)
		all[r.AwayTeamID] = append(all[r.AwayTeamID], a)
		leagueHome = append(leagueHome, h)
		leagueAway = append(leagueAway, a)
	}

	// The league's own average at each venue, which estimates pull towards (prior_games).
	for venue, results := range map[engine.Venue][]engine.Result{engine.Home: leagueHome, engine.Away: leagueAway} {
		p := s.engine.BuildLeagueProfile(league, venue, results)
		if err := q.UpsertLeagueProfile(ctx, dbsqlc.UpsertLeagueProfileParams{
			LeagueID: league, Venue: string(venue), SampleSize: int32(p.SampleSize),
			WinRate: p.WinRate, DrawRate: p.DrawRate, LossRate: p.LossRate,
			GfAvg: p.GoalsForAvg, GaAvg: p.GoalsAgainstAvg,
			ScoredRate: p.ScoredRate, CleanSheetRate: p.CleanSheetRate, BttsRate: p.BTTSRate,
			Over15Rate: p.Over15Rate, Over25Rate: p.Over25Rate, Under35Rate: p.Under35Rate,
		}); err != nil {
			return 0, nil, fmt.Errorf("save %s league profile: %w", league, err)
		}
	}

	window := int16(s.engine.Thresholds().FormWindow)
	count := 0
	var low []string
	for _, t := range teams {
		isLow := false
		for venue, results := range map[engine.Venue][]engine.Result{engine.Home: home[t.ID], engine.Away: away[t.ID]} {
			p := s.engine.BuildProfile(t.Name, venue, results)
			if s.engine.LowConfidence(p) {
				isLow = true
			}
			if err := q.UpsertTeamProfile(ctx, dbsqlc.UpsertTeamProfileParams{
				TeamID: t.ID, Venue: string(venue), WindowN: window, SampleSize: int16(p.SampleSize),
				WinRate: p.WinRate, DrawRate: p.DrawRate, LossRate: p.LossRate,
				GfAvg: p.GoalsForAvg, GaAvg: p.GoalsAgainstAvg,
				ScoredRate: p.ScoredRate, CleanSheetRate: p.CleanSheetRate, BttsRate: p.BTTSRate,
				Over15Rate: p.Over15Rate, Over25Rate: p.Over25Rate, Under35Rate: p.Under35Rate,
			}); err != nil {
				return count, low, fmt.Errorf("save profile for %s: %w", t.Name, err)
			}
			count++
		}
		if isLow {
			low = append(low, t.Name)
		}
		if err := q.UpdateTeamForm(ctx, dbsqlc.UpdateTeamFormParams{ID: t.ID, Form: engine.Form(all[t.ID])}); err != nil {
			return count, low, err
		}
	}
	sort.Strings(low)
	return count, low, nil
}

// teamResolver finds a club by normalised name or alias, creating it the first
// time the primary source mentions it (FR-1.5).
type teamResolver struct {
	q       *dbsqlc.Queries
	league  string
	byKey   map[string]int64
	created []string
}

func newTeamResolver(ctx context.Context, q *dbsqlc.Queries, league string) (*teamResolver, error) {
	teams, err := q.ListTeamsByLeague(ctx, league)
	if err != nil {
		return nil, err
	}
	r := &teamResolver{q: q, league: league, byKey: map[string]int64{}}
	for _, t := range teams {
		r.byKey[t.NameKey] = t.ID
		for _, alias := range t.Aliases {
			r.byKey[alias] = t.ID
		}
	}
	return r, nil
}

// find looks a club up by any of its names, without adding it.
func (r *teamResolver) find(names ...string) (int64, bool) {
	for _, n := range names {
		if key := teamnames.Key(n); key != "" {
			if id, ok := r.byKey[key]; ok {
				return id, true
			}
		}
	}
	return 0, false
}

func (r *teamResolver) resolve(ctx context.Context, name string) (int64, error) {
	key := teamnames.Key(name)
	if key == "" {
		return 0, fmt.Errorf("team name %q has no letters", name)
	}
	if id, ok := r.byKey[key]; ok {
		return id, nil
	}
	t, err := r.q.InsertTeam(ctx, dbsqlc.InsertTeamParams{LeagueID: r.league, Name: name, NameKey: key})
	if err != nil {
		return 0, fmt.Errorf("add team %q: %w", name, err)
	}
	r.byKey[key] = t.ID
	r.created = append(r.created, name)
	return t.ID, nil
}
