package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
)

// Saved tickets (V2, SRS addendum B, FR-9), without accounts. Decided
// 2026-11-27: kept 90 days after the last game settles; anyone with the link
// can view; only the browser that saved it (which holds the ticket_id) can
// update or delete; no notifications.

const (
	// SavedKeepDays is how long a ticket is kept after its last game settles.
	SavedKeepDays = 90
	// savedMaxDays removes tickets that never settle (a game cancelled for good).
	savedMaxDays = 180
	// MaxSavedLookup caps how many tickets one "My tickets" request can ask for.
	MaxSavedLookup = 50
)

var savedIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22}$`)

// Outcome of a pick: won, lost, or pending (not played, or postponed).
type Outcome string

const (
	Won     Outcome = "won"
	Lost    Outcome = "lost"
	Pending Outcome = "pending"
)

type SavedPick struct {
	Market   string   `json:"market"`
	Odds     *float64 `json:"odds,omitempty"`
	Estimate float64  `json:"estimate,omitempty"`
	RiskBand string   `json:"risk_band,omitempty"`
	Outcome  Outcome  `json:"outcome"`
}

type SavedSuggestion struct {
	Market   string  `json:"market"`
	Estimate float64 `json:"estimate"`
	RiskBand string  `json:"risk_band"`
	Accepted bool    `json:"accepted"`
	Outcome  Outcome `json:"outcome"`
}

type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

type SavedLeg struct {
	Fixture     Fixture           `json:"fixture"`
	MatchStatus string            `json:"match_status"`
	Score       *Score            `json:"score"`
	Original    SavedPick         `json:"original"`
	Final       SavedPick         `json:"final"`
	Suggestions []SavedSuggestion `json:"suggestions"`
}

// PairCounts compares, for every suggestion we showed, how it did against the
// user's own pick in the same game (FR-9.6). Misses count as much as hits (RG-3).
type PairCounts struct {
	SuggestionWonPickLost int `json:"suggestion_won_pick_lost"`
	PickWonSuggestionLost int `json:"pick_won_suggestion_lost"`
	BothWon               int `json:"both_won"`
	BothLost              int `json:"both_lost"`
}

type SavedSummary struct {
	Legs     int        `json:"legs"`
	Settled  int        `json:"settled"`
	Original HitCount   `json:"original"`
	Final    HitCount   `json:"final"`
	Pairs    PairCounts `json:"pairs"`
	// The whole ticket as first picked and as finally chosen: won only if
	// every leg won, lost as soon as one lost.
	OriginalTicket Outcome `json:"original_ticket"`
	FinalTicket    Outcome `json:"final_ticket"`
}

type SavedTicket struct {
	ID        string     `json:"id"`
	Origin    string     `json:"origin"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Status    Outcome    `json:"status"` // pending or settled ("won"/"lost" live in the summary)
	SettledAt *time.Time `json:"settled_at"`
	// ExpiresAt is when the ticket will be deleted; null until it settles.
	ExpiresAt *time.Time   `json:"expires_at"`
	Summary   SavedSummary `json:"summary"`
	// Legs are left out of list responses.
	Legs []SavedLeg `json:"legs,omitempty"`
}

const settled Outcome = "settled"

type SavedService struct {
	pool *pgxpool.Pool
	q    *dbsqlc.Queries
	now  func() time.Time
}

func NewSavedService(pool *pgxpool.Pool, now func() time.Time) *SavedService {
	return &SavedService{pool: pool, q: dbsqlc.New(pool), now: now}
}

// Save snapshots an analysed ticket (FR-9.1, FR-9.3). Saving the same ticket
// again replaces the snapshot and keeps its link.
func (s *SavedService) Save(ctx context.Context, ticketID string) (*SavedTicket, error) {
	if !ticketIDPattern.MatchString(ticketID) {
		return nil, Invalid("ticket_id must be 8–64 letters, digits or dashes.")
	}
	var id string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		legs, err := q.ListTicketLegs(ctx, ticketID)
		if err != nil {
			return err
		}
		origin := ""
		for _, l := range legs {
			if l.OnTicket {
				origin = l.Origin
				break
			}
		}
		if origin == "" {
			return NotFound("Analyse the ticket before saving it.")
		}

		existing, err := q.GetSavedByTicket(ctx, ticketID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			id = newSavedID()
			err = q.InsertSaved(ctx, dbsqlc.InsertSavedParams{ID: id, TicketID: ticketID, Origin: origin})
		case err == nil:
			id = existing.ID
			if err = q.ResaveSaved(ctx, dbsqlc.ResaveSavedParams{ID: id, Origin: origin}); err == nil {
				err = q.DeleteSavedLegs(ctx, id)
			}
		}
		if err != nil {
			return err
		}
		_, err = q.SnapshotLegs(ctx, dbsqlc.SnapshotLegsParams{SavedID: id, TicketID: ticketID})
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Get returns one saved ticket with every leg (FR-9.2, FR-9.5).
func (s *SavedService) Get(ctx context.Context, id string) (*SavedTicket, error) {
	if !savedIDPattern.MatchString(id) {
		return nil, NotFound("No saved ticket with that link. Saved tickets are kept for 90 days after their last game.")
	}
	list, err := s.load(ctx, []string{id}, true)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, NotFound("No saved ticket with that link. Saved tickets are kept for 90 days after their last game.")
	}
	return &list[0], nil
}

// List returns summaries for the tickets a browser has saved (FR-9.8); ids
// that no longer exist (deleted or expired) are left out, so the browser can
// forget them.
func (s *SavedService) List(ctx context.Context, ids []string) ([]SavedTicket, error) {
	if len(ids) > MaxSavedLookup {
		return nil, Invalid(fmt.Sprintf("Ask for at most %d tickets at once.", MaxSavedLookup))
	}
	valid := ids[:0:0]
	for _, id := range ids {
		if savedIDPattern.MatchString(id) {
			valid = append(valid, id)
		}
	}
	return s.load(ctx, valid, false)
}

// Delete removes a saved ticket (FR-9.10). It needs the ticket's own id, which
// only the browser that saved it has; the link alone isn't enough. The ticket
// log (and so the public track record) is not affected.
func (s *SavedService) Delete(ctx context.Context, id, ticketID string) error {
	n, err := s.q.DeleteSaved(ctx, dbsqlc.DeleteSavedParams{ID: id, TicketID: ticketID})
	if err == nil && n == 0 {
		return NotFound("No saved ticket of yours with that link.")
	}
	return err
}

// Cleanup marks tickets whose games have all been played as settled, and
// deletes those settled more than 90 days ago (and any older than 180 days).
// The daily import runs it.
func (s *SavedService) Cleanup(ctx context.Context) (settledNow, deleted int64, err error) {
	if settledNow, err = s.q.MarkSavedSettled(ctx); err != nil {
		return 0, 0, err
	}
	now := s.now()
	deleted, err = s.q.DeleteExpiredSaved(ctx, dbsqlc.DeleteExpiredSavedParams{
		SettledBefore: now.AddDate(0, 0, -SavedKeepDays),
		CreatedBefore: now.AddDate(0, 0, -savedMaxDays),
	})
	return settledNow, deleted, err
}

func (s *SavedService) load(ctx context.Context, ids []string, withLegs bool) ([]SavedTicket, error) {
	out := []SavedTicket{}
	if len(ids) == 0 {
		return out, nil
	}
	tickets, err := s.q.ListSaved(ctx, ids)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListSavedLegs(ctx, ids)
	if err != nil {
		return nil, err
	}
	legs := map[string][]SavedLeg{}
	for _, r := range rows {
		l, err := savedLeg(r)
		if err != nil {
			return nil, err
		}
		legs[r.SavedTicketID] = append(legs[r.SavedTicketID], l)
	}

	// Keep the order the browser asked in (newest saved first, usually).
	byID := map[string]dbsqlc.SavedTicket{}
	for _, t := range tickets {
		byID[t.ID] = t
	}
	for _, id := range ids {
		t, ok := byID[id]
		if !ok {
			continue
		}
		st := SavedTicket{ID: t.ID, Origin: t.Origin, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, Summary: summarise(legs[id])}
		st.Status = Pending
		if st.Summary.Legs > 0 && st.Summary.Settled == st.Summary.Legs {
			st.Status = settled
			// Settled before the daily clean-up noticed: count from now.
			at := s.now()
			if t.SettledAt != nil {
				at = *t.SettledAt
			}
			exp := at.AddDate(0, 0, SavedKeepDays)
			st.SettledAt, st.ExpiresAt = &at, &exp
		}
		if withLegs {
			st.Legs = legs[id]
		}
		out = append(out, st)
	}
	return out, nil
}

func savedLeg(r dbsqlc.ListSavedLegsRow) (SavedLeg, error) {
	l := SavedLeg{
		Fixture:     fixture(r.MatchID, r.LeagueID, r.Kickoff, r.HomeID, r.HomeName, r.HomeForm, r.AwayID, r.AwayName, r.AwayForm),
		MatchStatus: r.Status,
		Original:    SavedPick{Market: r.OriginalMarket, Odds: r.OriginalOdds, Estimate: r.OriginalEstimate, RiskBand: r.OriginalBand},
		Final:       SavedPick{Market: r.FinalMarket},
		Suggestions: []SavedSuggestion{},
	}
	if err := json.Unmarshal(r.Suggestions, &l.Suggestions); err != nil {
		return l, fmt.Errorf("saved suggestions: %w", err)
	}
	finished := r.Status == "finished" && r.HomeGoals != nil && r.AwayGoals != nil
	if finished {
		l.Score = &Score{Home: int(*r.HomeGoals), Away: int(*r.AwayGoals)}
	}
	outcome := func(market string) (Outcome, error) {
		if !finished {
			return Pending, nil
		}
		o, err := engine.Settle(engine.Market(market), l.Score.Home, l.Score.Away)
		if err != nil {
			return "", err
		}
		if o == engine.Won {
			return Won, nil
		}
		return Lost, nil
	}
	var err error
	if l.Original.Outcome, err = outcome(l.Original.Market); err != nil {
		return l, err
	}
	if l.Final.Outcome, err = outcome(l.Final.Market); err != nil {
		return l, err
	}
	for i := range l.Suggestions {
		if l.Suggestions[i].Outcome, err = outcome(l.Suggestions[i].Market); err != nil {
			return l, err
		}
	}
	return l, nil
}

func summarise(legs []SavedLeg) SavedSummary {
	sum := SavedSummary{Legs: len(legs), OriginalTicket: Pending, FinalTicket: Pending}
	if len(legs) == 0 {
		return sum
	}
	count := func(h *HitCount, o Outcome) {
		switch o {
		case Won:
			h.Won++
		case Lost:
			h.Lost++
		}
	}
	for _, l := range legs {
		if l.Original.Outcome == Pending {
			continue
		}
		sum.Settled++
		count(&sum.Original, l.Original.Outcome)
		count(&sum.Final, l.Final.Outcome)
		for _, sg := range l.Suggestions {
			switch {
			case sg.Outcome == Won && l.Original.Outcome == Lost:
				sum.Pairs.SuggestionWonPickLost++
			case sg.Outcome == Lost && l.Original.Outcome == Won:
				sum.Pairs.PickWonSuggestionLost++
			case sg.Outcome == Won:
				sum.Pairs.BothWon++
			default:
				sum.Pairs.BothLost++
			}
		}
	}
	ticket := func(h HitCount) Outcome {
		switch {
		case h.Lost > 0:
			return Lost
		case h.Won == len(legs):
			return Won
		}
		return Pending
	}
	sum.OriginalTicket, sum.FinalTicket = ticket(sum.Original), ticket(sum.Final)
	return sum
}

// newSavedID is 128 random bits: the private link, so it must not be guessable.
func newSavedID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
