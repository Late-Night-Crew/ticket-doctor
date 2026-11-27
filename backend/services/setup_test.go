package services

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
)

// testNow is a Friday at noon in CAT (10:00 UTC).
var testNow = time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)

func clock() time.Time { return testNow }

// rates are the numbers stored in a team profile.
type rates struct {
	win, draw, loss, scored, clean, btts, o15, o25, u35 float64
	sample                                              int16
}

// goalsHeavy: wins are rare (High risk) but goals are common, so a win pick
// gets goal-market suggestions.
var goalsHeavy = rates{win: 0.3, draw: 0.3, loss: 0.4, scored: 0.9, clean: 0.1, btts: 0.8, o15: 0.9, o25: 0.7, u35: 0.4, sample: 10}

type world struct {
	pool  *pgxpool.Pool
	q     *dbsqlc.Queries
	teams map[string]int64
	// matches by "Home v Away"
	matches map[string]int64
}

// newWorld makes four EPL teams with full profiles and these matches.
func newWorld(t *testing.T, pool *pgxpool.Pool) *world {
	t.Helper()
	ctx := context.Background()
	w := &world{pool: pool, q: dbsqlc.New(pool), teams: map[string]int64{}, matches: map[string]int64{}}
	for _, name := range []string{"Arsenal", "Chelsea", "Everton", "Leeds"} {
		team, err := w.q.InsertTeam(ctx, dbsqlc.InsertTeamParams{LeagueID: "epl", Name: name, NameKey: name})
		if err != nil {
			t.Fatal(err)
		}
		w.teams[name] = team.ID
		if _, err := pool.Exec(ctx, `UPDATE teams SET form = 'W-D-L-W-W' WHERE id = $1`, team.ID); err != nil {
			t.Fatal(err)
		}
		for _, venue := range []string{"home", "away"} {
			w.profile(t, name, venue, goalsHeavy)
		}
	}
	return w
}

func (w *world) profile(t *testing.T, team, venue string, r rates) {
	t.Helper()
	err := w.q.UpsertTeamProfile(context.Background(), dbsqlc.UpsertTeamProfileParams{
		TeamID: w.teams[team], Venue: venue, WindowN: 10, SampleSize: r.sample,
		WinRate: r.win, DrawRate: r.draw, LossRate: r.loss, GfAvg: 1.5, GaAvg: 1.2,
		ScoredRate: r.scored, CleanSheetRate: r.clean, BttsRate: r.btts,
		Over15Rate: r.o15, Over25Rate: r.o25, Under35Rate: r.u35,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// match adds a scheduled match and returns its id as the API sends it.
func (w *world) match(t *testing.T, home, away string, kickoff time.Time) string {
	t.Helper()
	row, err := w.q.UpsertFixture(context.Background(), dbsqlc.UpsertFixtureParams{
		LeagueID: "epl", Season: "2627", Kickoff: kickoff,
		HomeTeamID: w.teams[home], AwayTeamID: w.teams[away], Status: "scheduled", Source: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	w.matches[home+" v "+away] = row.ID
	return strconv.FormatInt(row.ID, 10)
}

// finish records a final score.
func (w *world) finish(t *testing.T, matchID string, home, away int) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(),
		`UPDATE matches SET status = 'finished', home_goals = $2, away_goals = $3 WHERE id = $1`,
		matchID, home, away); err != nil {
		t.Fatal(err)
	}
}

func (w *world) set(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func estimateFor(t *testing.T, m engine.Market) engine.MarketEstimate {
	t.Helper()
	p := func(v engine.Venue) engine.Profile {
		r := goalsHeavy
		return engine.Profile{Venue: v, SampleSize: int(r.sample), WinRate: r.win, DrawRate: r.draw, LossRate: r.loss,
			GoalsForAvg: 1.5, GoalsAgainstAvg: 1.2, ScoredRate: r.scored, CleanSheetRate: r.clean, BTTSRate: r.btts,
			Over15Rate: r.o15, Over25Rate: r.o25, Under35Rate: r.u35}
	}
	est, err := engine.Default().Estimate(m, p(engine.Home), p(engine.Away))
	if err != nil {
		t.Fatal(err)
	}
	return est
}
