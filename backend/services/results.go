package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
)

type HitCount struct {
	Won  int `json:"won"`
	Lost int `json:"lost"`
}

func (h *HitCount) add(o engine.Outcome) {
	if o == engine.Won {
		h.Won++
	} else {
		h.Lost++
	}
}

type ResultsRow struct {
	LeagueID string `json:"league_id"`
	Market   string `json:"market"`
	// Original counts the users' own picks in this market.
	Original HitCount `json:"original"`
	// Suggested counts our suggestions in this market.
	Suggested HitCount `json:"suggested"`
	// OriginalWithSuggestion counts the users' picks in this market that we
	// suggested a swap for: the like-for-like comparison (addendum A6, FR-7.3a).
	OriginalWithSuggestion HitCount `json:"original_with_suggestion"`
}

type ResultsResponse struct {
	// UpdatedAt is when results were last imported; null before the first import.
	UpdatedAt *time.Time   `json:"updated_at"`
	Rows      []ResultsRow `json:"rows"`
}

// ResultsService builds the public track record (FR-7.2, FR-7.3). Outcomes
// are worked out from the final score when read, so a newly imported matchday
// shows up straight away and unplayed matches are never counted.
type ResultsService struct {
	q *dbsqlc.Queries
}

func NewResultsService(db dbsqlc.DBTX) *ResultsService {
	return &ResultsService{q: dbsqlc.New(db)}
}

// Results returns one row per league and market with anything settled,
// optionally filtered. Rows are in league order, then canonical market order.
func (s *ResultsService) Results(ctx context.Context, league, market string) (*ResultsResponse, error) {
	if market != "" && !engine.Market(market).Valid() {
		return nil, Invalid(fmt.Sprintf("Unknown market %q.", market))
	}
	leagues, err := s.q.ListLeagues(ctx)
	if err != nil {
		return nil, err
	}
	leagueOrder := map[string]int{}
	for i, l := range leagues {
		leagueOrder[l.ID] = i
	}
	if _, ok := leagueOrder[league]; league != "" && !ok {
		return nil, NotFound(fmt.Sprintf("No league called %q.", league))
	}
	keep := func(l, m string) bool { return (league == "" || l == league) && (market == "" || m == market) }

	type key struct{ league, market string }
	rows := map[key]*ResultsRow{}
	row := func(l, m string) *ResultsRow {
		k := key{l, m}
		if rows[k] == nil {
			rows[k] = &ResultsRow{LeagueID: l, Market: m}
		}
		return rows[k]
	}

	legs, err := s.q.ListSettledLegs(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range legs {
		if !keep(l.LeagueID, l.OriginalMarket) {
			continue
		}
		o, err := settle(l.OriginalMarket, l.HomeGoals, l.AwayGoals)
		if err != nil {
			return nil, err
		}
		r := row(l.LeagueID, l.OriginalMarket)
		r.Original.add(o)
		if l.HadSuggestion {
			r.OriginalWithSuggestion.add(o)
		}
	}
	suggestions, err := s.q.ListSettledSuggestions(ctx)
	if err != nil {
		return nil, err
	}
	for _, sg := range suggestions {
		if !keep(sg.LeagueID, sg.Market) {
			continue
		}
		o, err := settle(sg.Market, sg.HomeGoals, sg.AwayGoals)
		if err != nil {
			return nil, err
		}
		row(sg.LeagueID, sg.Market).Suggested.add(o)
	}

	res := &ResultsResponse{Rows: make([]ResultsRow, 0, len(rows))}
	for _, r := range rows {
		res.Rows = append(res.Rows, *r)
	}
	slices.SortFunc(res.Rows, func(a, b ResultsRow) int {
		if d := leagueOrder[a.LeagueID] - leagueOrder[b.LeagueID]; d != 0 {
			return d
		}
		return slices.Index(engine.Markets, engine.Market(a.Market)) - slices.Index(engine.Markets, engine.Market(b.Market))
	})

	res.UpdatedAt, err = s.q.LastSuccessfulImport(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return res, err
}

func settle(market string, home, away *int16) (engine.Outcome, error) {
	if home == nil || away == nil {
		return "", fmt.Errorf("finished match without a score")
	}
	return engine.Settle(engine.Market(market), int(*home), int(*away))
}
