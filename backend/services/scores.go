package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/logging/loggedmodule"
)

// Quick score checks (addendum A10). The results file, football-data.co.uk,
// posts results a day or more after the games, so a saved ticket would wait
// that long to settle. On match days a light job asks football-data.org, every
// 15 minutes, for games that have kicked off and have no result yet, and saves
// their full-time scores. The full import still runs twice a day, and the
// results file replaces these scores when it arrives.

const (
	// ScoresLookback is how far back a check looks for games still without a
	// result, so a game missed one evening is picked up the next day.
	ScoresLookback = 36 * time.Hour
	// ScoresMinAge skips games kicked off too recently to be over.
	ScoresMinAge = 100 * time.Minute
	// scoresKeep is how long score-check runs stay in import_runs.
	scoresKeep = 14 * 24 * time.Hour
)

// LeagueScores is what a score check did for one league.
type LeagueScores struct {
	League string `json:"league"`
	// Rows is how many games football-data.org returned.
	Rows int `json:"rows"`
	// Results counts games newly given a full-time score.
	Results   int      `json:"results"`
	Profiles  int      `json:"profiles"`
	Unmatched []string `json:"unmatched,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// ScoresSummary is stored on the import_runs row (kind "scores").
type ScoresSummary struct {
	// RunID is 0 when there was nothing to check, and nothing was recorded.
	RunID   int64          `json:"run_id"`
	Leagues []LeagueScores `json:"leagues"`
}

// Results is the number of games given a score in this check.
func (s *ScoresSummary) Results() int {
	n := 0
	for _, l := range s.Leagues {
		n += l.Results
	}
	return n
}

// QuickScores runs one score check at now. With no game awaiting a result it
// makes no request and records nothing. It needs a fixtures source.
func (s *ImportService) QuickScores(ctx context.Context, now time.Time) (*ScoresSummary, error) {
	if s.fixtures == nil {
		return nil, errors.New("score checks need a football-data.org source")
	}
	leagues, err := s.q.ListAwaitingScoreLeagues(ctx, dbsqlc.ListAwaitingScoreLeaguesParams{
		FromTime: now.Add(-ScoresLookback), ToTime: now.Add(-ScoresMinAge),
	})
	if err != nil {
		return nil, err
	}
	summary := &ScoresSummary{Leagues: []LeagueScores{}}
	if len(leagues) == 0 {
		return summary, nil
	}

	runID, err := s.q.StartImportRun(ctx, "scores")
	if err != nil {
		return nil, fmt.Errorf("record score check: %w", err)
	}
	summary.RunID = runID

	var failed []string
	for _, l := range leagues {
		ls, err := s.leagueScores(ctx, l.ID, l.FdoCode, l.Earliest, now)
		if err != nil {
			ls.Error = err.Error()
			failed = append(failed, l.ID)
			s.logger.Error("score check failed", loggedmodule.Attr(loggedmodule.Import), "league", l.ID, "error", err)
		}
		summary.Leagues = append(summary.Leagues, ls)
	}

	status, errText := "succeeded", (*string)(nil)
	var runErr error
	if len(failed) > 0 {
		status = "failed"
		runErr = fmt.Errorf("score check failed for %v", failed)
		msg := runErr.Error()
		errText = &msg
	}
	body, _ := json.Marshal(summary)
	ctx = context.WithoutCancel(ctx)
	if err := s.q.FinishImportRun(ctx, dbsqlc.FinishImportRunParams{
		ID: runID, Status: status, Summary: body, Error: errText,
	}); err != nil {
		return summary, errors.Join(runErr, fmt.Errorf("record score check result: %w", err))
	}
	if _, err := s.q.DeleteOldScoreRuns(ctx, now.Add(-scoresKeep)); err != nil {
		return summary, errors.Join(runErr, fmt.Errorf("prune old score checks: %w", err))
	}
	return summary, runErr
}

func (s *ImportService) leagueScores(ctx context.Context, league, code string, earliest, now time.Time) (LeagueScores, error) {
	ls := LeagueScores{League: league}
	// One request covers every day from the earliest game awaiting a result.
	fixtures, err := s.fixtures.Fixtures(ctx, code, earliest.UTC().Truncate(24*time.Hour), now.UTC().Truncate(24*time.Hour))
	if err != nil {
		return ls, err
	}
	ls.Rows = len(fixtures)

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		teams, err := newTeamResolver(ctx, q, league)
		if err != nil {
			return err
		}
		unmatched := map[string]bool{}
		for _, f := range fixtures {
			if f.Score == nil {
				continue
			}
			for _, t := range [][2]string{{f.Home.Name, f.Home.ShortName}, {f.Away.Name, f.Away.ShortName}} {
				if _, ok := teams.find(t[0], t[1]); !ok {
					unmatched[t[0]] = true
				}
			}
			saved, err := saveScore(ctx, q, league, teams, f)
			if err != nil {
				return err
			}
			if saved {
				ls.Results++
			}
		}
		for name := range unmatched {
			ls.Unmatched = append(ls.Unmatched, name)
		}
		sort.Strings(ls.Unmatched)
		if ls.Results == 0 {
			return nil
		}
		ls.Profiles, _, err = s.rebuildProfiles(ctx, q, league)
		return err
	})
	if err != nil {
		ls.Results, ls.Profiles = 0, 0
		return ls, err
	}
	s.logger.Info("scores checked", loggedmodule.Attr(loggedmodule.Import),
		"league", league, "rows", ls.Rows, "results", ls.Results, "unmatched", ls.Unmatched)
	if len(ls.Unmatched) > 0 {
		return ls, fmt.Errorf("no team matches %q; add an alias in pkg/teamnames (the other scores were saved)", ls.Unmatched)
	}
	return ls, nil
}
