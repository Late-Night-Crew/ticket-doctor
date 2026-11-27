package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
)

// AdminService backs the admin pages (B6): the dashboard, manual result entry
// (FR-1.4) and the rule thresholds view.
type AdminService struct {
	pool *pgxpool.Pool
	q    *dbsqlc.Queries
	t    engine.Thresholds
	now  func() time.Time
}

func NewAdminService(pool *pgxpool.Pool, t engine.Thresholds, now func() time.Time) *AdminService {
	return &AdminService{pool: pool, q: dbsqlc.New(pool), t: t, now: now}
}

// ---------- dashboard ----------

type ImportRun struct {
	ID         int64           `json:"id,string"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt *time.Time      `json:"finished_at"`
	Status     string          `json:"status"`
	Summary    json.RawMessage `json:"summary"`
	Error      *string         `json:"error"`
}

type Dashboard struct {
	Matches struct {
		Upcoming       int64 `json:"upcoming"`
		AwaitingResult int64 `json:"awaiting_result"`
		Finished       int64 `json:"finished"`
	} `json:"matches"`
	Tickets struct {
		Total    int64 `json:"total"`
		LastWeek int64 `json:"last_7_days"`
		Legs     int64 `json:"legs"`
	} `json:"tickets"`
	Suggestions struct {
		Shown    int64 `json:"shown"`
		Accepted int64 `json:"accepted"`
	} `json:"suggestions"`
	Imports []ImportRun `json:"imports"`
}

// Dashboard sums up the data and the last ten import runs.
func (s *AdminService) Dashboard(ctx context.Context) (*Dashboard, error) {
	c, err := s.q.DashboardCounts(ctx, s.now())
	if err != nil {
		return nil, err
	}
	d := &Dashboard{Imports: []ImportRun{}}
	d.Matches.Upcoming, d.Matches.AwaitingResult, d.Matches.Finished = c.Upcoming, c.AwaitingResult, c.Finished
	d.Tickets.Total, d.Tickets.LastWeek, d.Tickets.Legs = c.Tickets, c.Tickets7d, c.Legs
	d.Suggestions.Shown, d.Suggestions.Accepted = c.Suggestions, c.Accepted

	runs, err := s.q.ListImportRuns(ctx, 10)
	if err != nil {
		return nil, err
	}
	for _, r := range runs {
		d.Imports = append(d.Imports, ImportRun{ID: r.ID, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
			Status: r.Status, Summary: json.RawMessage(r.Summary), Error: r.Error})
	}
	return d, nil
}

// ---------- result entry ----------

type AdminMatch struct {
	ID        string    `json:"id"`
	LeagueID  string    `json:"league_id"`
	Kickoff   time.Time `json:"kickoff"`
	Status    string    `json:"status"`
	Home      string    `json:"home"`
	Away      string    `json:"away"`
	HomeGoals *int16    `json:"home_goals"`
	AwayGoals *int16    `json:"away_goals"`
	Source    string    `json:"source"`
}

// Matches lists games for the result entry page: "awaiting" (kicked off, no
// result yet; oldest first) or "recent" (finished in the last two weeks;
// newest first, to correct a score). league is optional.
func (s *AdminService) Matches(ctx context.Context, view, league string) ([]AdminMatch, error) {
	var awaiting bool
	switch view {
	case "", "awaiting":
		awaiting = true
	case "recent":
	default:
		return nil, Invalid(fmt.Sprintf("view must be awaiting or recent, not %q", view))
	}
	p := dbsqlc.ListAdminMatchesParams{Awaiting: awaiting, Now: s.now()}
	if league != "" {
		p.LeagueID = &league
	}
	rows, err := s.q.ListAdminMatches(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]AdminMatch, 0, len(rows))
	for _, r := range rows {
		out = append(out, AdminMatch{ID: strconv.FormatInt(r.ID, 10), LeagueID: r.LeagueID, Kickoff: r.Kickoff,
			Status: r.Status, Home: r.HomeName, Away: r.AwayName, HomeGoals: r.HomeGoals, AwayGoals: r.AwayGoals, Source: r.Source})
	}
	return out, nil
}

// maxGoals is far above any real score, but catches a typo like 22 for 2.
const maxGoals = 20

// EnterResult records a final score typed in by an admin (FR-1.4): the match
// becomes finished, so the track record counts it straight away, and the entry
// is kept in result_entries. The results import still wins for leagues it
// covers: if its file later has a different score for this match, the file's
// score replaces this one. Team profiles pick the result up at the next import.
func (s *AdminService) EnterResult(ctx context.Context, by *Admin, matchID string, home, away int) error {
	id, err := strconv.ParseInt(matchID, 10, 64)
	if err != nil {
		return NotFound(fmt.Sprintf("No match with id %q.", matchID))
	}
	if home < 0 || away < 0 || home > maxGoals || away > maxGoals {
		return Invalid(fmt.Sprintf("Goals must be between 0 and %d.", maxGoals))
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		m, err := q.GetMatchForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return NotFound(fmt.Sprintf("No match with id %q.", matchID))
		}
		if err != nil {
			return err
		}
		if m.Kickoff.After(s.now()) {
			return Invalid("This match hasn't kicked off yet.")
		}
		h, a := int16(home), int16(away)
		if err := q.SetManualResult(ctx, dbsqlc.SetManualResultParams{ID: id, HomeGoals: &h, AwayGoals: &a}); err != nil {
			return err
		}
		return q.InsertResultEntry(ctx, dbsqlc.InsertResultEntryParams{MatchID: id, UserID: by.ID,
			HomeGoals: h, AwayGoals: a, PreviousHome: m.HomeGoals, PreviousAway: m.AwayGoals})
	})
}

// ---------- thresholds ----------

type Threshold struct {
	// Key is the name under "rules:" in config.yaml.
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
	Default     float64 `json:"default"`
	// Unit is "percent" (a 0–1 value shown as a percentage) or "count".
	Unit string `json:"unit"`
}

// Thresholds lists the rule thresholds in use, with the SRS defaults. They
// are read-only here: they change in config.yaml (or the release), so every
// change is reviewed and recorded (NFR-6).
func (s *AdminService) Thresholds() []Threshold {
	cur, def := s.t, engine.DefaultThresholds()
	pct := func(key, label, desc string, v, d float64) Threshold {
		return Threshold{Key: key, Label: label, Description: desc, Value: v, Default: d, Unit: "percent"}
	}
	cnt := func(key, label, desc string, v, d int) Threshold {
		return Threshold{Key: key, Label: label, Description: desc, Value: float64(v), Default: float64(d), Unit: "count"}
	}
	return []Threshold{
		cnt("form_window", "Form window", "How many recent home or away games a team profile uses (FR-2.1).", cur.FormWindow, def.FormWindow),
		cnt("low_confidence_below", "Low confidence below", "Profiles built from fewer games are marked low confidence (FR-2.3).", cur.LowConfidenceBelow, def.LowConfidenceBelow),
		pct("low_from", "Low risk from", "Estimates at or above this are Low risk (4.2).", cur.LowFrom, def.LowFrom),
		pct("medium_from", "Medium risk from", "Estimates at or above this, but below Low, are Medium risk; anything lower is High (4.2).", cur.MediumFrom, def.MediumFrom),
		pct("swap_min_gain", "Swap minimum gain", "A swap is only suggested if its estimate is at least this much higher (4.3).", cur.SwapMinGain, def.SwapMinGain),
		cnt("max_suggestions", "Suggestions per leg", "The most swaps offered for one leg (4.3).", cur.MaxSuggestions, def.MaxSuggestions),
		pct("value_margin", "Value margin", "A pick is marked value when its estimate beats the odds' implied chance by this much (4.5).", cur.ValueMargin, def.ValueMargin),
		pct("bold_floor", "Bold floor", "The lowest estimate the ticket wizard will ever use (4.6).", cur.BoldFloor, def.BoldFloor),
		pct("max_estimate", "Estimate cap", "No estimate goes above this, so nothing looks certain (addendum A7).", cur.MaxEstimate, def.MaxEstimate),
	}
}
