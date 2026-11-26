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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
	"ticketdoctor/logging/loggedmodule"
	"ticketdoctor/pkg/footballdata"
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

// ImportOptions chooses what to import. Empty Leagues means every league with
// a football-data.co.uk code.
type ImportOptions struct {
	Leagues []string
	Seasons []string
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
	Profiles      int      `json:"profiles"`
	// LowConfidence lists teams with a home or away profile below the cut-off.
	LowConfidence []string `json:"low_confidence,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// ImportSummary is stored on the import_runs row and logged.
type ImportSummary struct {
	RunID   int64          `json:"run_id"`
	Leagues []LeagueImport `json:"leagues"`
}

// ImportService loads results into the database and rebuilds team profiles
// (FR-1.1, FR-1.3, FR-1.5, FR-2).
type ImportService struct {
	pool   *pgxpool.Pool
	q      *dbsqlc.Queries
	source ResultsSource
	engine *engine.Engine
	logger *slog.Logger
}

func NewImportService(pool *pgxpool.Pool, source ResultsSource, eng *engine.Engine, logger *slog.Logger) *ImportService {
	return &ImportService{pool: pool, q: dbsqlc.New(pool), source: source, engine: eng, logger: logger}
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
	leagues, err := s.leagues(ctx, opts.Leagues)
	if err != nil {
		return nil, err
	}

	runID, err := s.q.StartImportRun(ctx)
	if err != nil {
		return nil, fmt.Errorf("record import run: %w", err)
	}
	summary := &ImportSummary{RunID: runID}

	var failed []string
	for _, l := range leagues {
		li, err := s.importLeague(ctx, l, opts.Seasons)
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

func (s *ImportService) importLeague(ctx context.Context, l dbsqlc.League, seasons []string) (LeagueImport, error) {
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

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		teams, err := newTeamResolver(ctx, q, l.ID)
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

		profiles, low, err := s.rebuildProfiles(ctx, q, l.ID)
		li.Profiles, li.LowConfidence = profiles, low
		return err
	})
	if err != nil {
		// The transaction rolled back: nothing from this league was saved.
		li.Inserted, li.Updated, li.Unchanged, li.NewTeams, li.Profiles, li.LowConfidence = 0, 0, 0, nil, 0, nil
		return li, err
	}
	s.logger.Info("league imported", loggedmodule.Attr(loggedmodule.Import),
		"league", l.ID, "rows", li.Rows, "inserted", li.Inserted, "updated", li.Updated,
		"unchanged", li.Unchanged, "skipped_rows", li.SkippedRows, "new_teams", len(li.NewTeams),
		"profiles", li.Profiles, "low_confidence_teams", len(li.LowConfidence))
	return li, nil
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
	for _, r := range rows {
		hg, ag := int(*r.HomeGoals), int(*r.AwayGoals)
		h := engine.Result{Date: r.Kickoff, GoalsFor: hg, GoalsAgainst: ag}
		a := engine.Result{Date: r.Kickoff, GoalsFor: ag, GoalsAgainst: hg}
		home[r.HomeTeamID] = append(home[r.HomeTeamID], h)
		away[r.AwayTeamID] = append(away[r.AwayTeamID], a)
		all[r.HomeTeamID] = append(all[r.HomeTeamID], h)
		all[r.AwayTeamID] = append(all[r.AwayTeamID], a)
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
