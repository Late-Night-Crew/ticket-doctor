package services

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
)

// Ticket limits (SRS FR-3; odds as the frontend's odds input allows).
const (
	MaxLegs = 15
	MinOdds = 1.01
	MaxOdds = 1000
)

var ticketIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// ---------- request and response (match frontend/src/api/types.ts) ----------

type TicketLegInput struct {
	MatchID string   `json:"match_id"`
	Market  string   `json:"market"`
	Odds    *float64 `json:"odds"`
}

type AnalyseRequest struct {
	TicketID string           `json:"ticket_id"`
	Origin   string           `json:"origin"`
	Legs     []TicketLegInput `json:"legs"`
}

type Suggestion struct {
	ID       string  `json:"id"`
	Market   string  `json:"market"`
	Estimate float64 `json:"estimate"`
	RiskBand string  `json:"risk_band"`
	Reason   string  `json:"reason"`
}

type AnalysedLeg struct {
	ID            string       `json:"id"`
	MatchID       string       `json:"match_id"`
	Fixture       Fixture      `json:"fixture"`
	Market        string       `json:"market"`
	Odds          *float64     `json:"odds"`
	Estimate      float64      `json:"estimate"`
	RiskBand      string       `json:"risk_band"`
	Reason        string       `json:"reason"`
	SampleSize    int          `json:"sample_size"`
	WindowN       int          `json:"window_n"`
	LowConfidence bool         `json:"low_confidence"`
	ImpliedProb   *float64     `json:"implied_prob"`
	IsValue       *bool        `json:"is_value"`
	Suggestions   []Suggestion `json:"suggestions"`
}

type CombinedProbability struct {
	Probability float64  `json:"probability"`
	OneIn       int      `json:"one_in"`
	Odds        *float64 `json:"odds"`
}

type AnalyseResponse struct {
	TicketID     string              `json:"ticket_id"`
	Legs         []AnalysedLeg       `json:"legs"`
	Combined     CombinedProbability `json:"combined"`
	WeakestLegID *string             `json:"weakest_leg_id"`
}

// ---------- service ----------

// TicketService analyses tickets (FR-4 to FR-6) and keeps the ticket log
// behind the track record (FR-7.1, SRS addendum A6).
type TicketService struct {
	pool   *pgxpool.Pool
	q      *dbsqlc.Queries
	engine *engine.Engine
	now    func() time.Time
}

func NewTicketService(pool *pgxpool.Pool, eng *engine.Engine, now func() time.Time) *TicketService {
	return &TicketService{pool: pool, q: dbsqlc.New(pool), engine: eng, now: now}
}

// legWork is one leg while it is being analysed.
type legWork struct {
	in      TicketLegInput
	matchID int64
	market  engine.Market
	odds    *float64
	match   dbsqlc.GetMatchesByIDsRow
	home    engine.Profile
	away    engine.Profile
	est     engine.MarketEstimate
	swaps   []engine.MarketEstimate
}

// Analyse rates each leg, suggests swaps and logs the ticket. Re-analysing the
// same ticket updates its log instead of adding to it.
func (s *TicketService) Analyse(ctx context.Context, req AnalyseRequest) (*AnalyseResponse, error) {
	legs, err := s.validate(req)
	if err != nil {
		return nil, err
	}
	if err := s.loadMatches(ctx, legs); err != nil {
		return nil, err
	}
	if err := s.loadProfiles(ctx, legs); err != nil {
		return nil, err
	}
	for _, l := range legs {
		l.est, err = s.engine.Estimate(l.market, l.home, l.away)
		if err != nil {
			return nil, err // markets were validated, so this is a bug
		}
		l.swaps = s.engine.Swaps(l.est, l.home, l.away)
	}

	res := &AnalyseResponse{TicketID: req.TicketID}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		out, err := s.log(ctx, s.q.WithTx(tx), req, legs)
		res.Legs = out
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("log ticket: %w", err)
	}
	res.Combined, res.WeakestLegID = s.combine(res.Legs)
	return res, nil
}

func (s *TicketService) validate(req AnalyseRequest) ([]*legWork, error) {
	if !ticketIDPattern.MatchString(req.TicketID) {
		return nil, Invalid("This ticket has no valid id. Reload the page and try again.")
	}
	if req.Origin != "manual" && req.Origin != "wizard" {
		return nil, Invalid("origin must be manual or wizard.")
	}
	if len(req.Legs) == 0 || len(req.Legs) > MaxLegs {
		return nil, Invalid(fmt.Sprintf("A ticket holds 1–%d legs.", MaxLegs))
	}
	seen := map[int64]bool{}
	legs := make([]*legWork, 0, len(req.Legs))
	for _, in := range req.Legs {
		id, err := strconv.ParseInt(in.MatchID, 10, 64)
		if err != nil || id <= 0 {
			return nil, NotFound(fmt.Sprintf("Unknown match %q.", in.MatchID))
		}
		if seen[id] {
			return nil, Invalid("Only one leg per match.")
		}
		seen[id] = true
		m := engine.Market(in.Market)
		if !m.Valid() {
			return nil, Invalid(fmt.Sprintf("Unknown market %q.", in.Market))
		}
		var odds *float64
		if in.Odds != nil {
			if math.IsNaN(*in.Odds) || *in.Odds < MinOdds || *in.Odds > MaxOdds {
				return nil, Invalid(fmt.Sprintf("Odds must be between %.2f and %.0f.", MinOdds, float64(MaxOdds)))
			}
			o := math.Round(*in.Odds*100) / 100 // stored to 2 decimals
			odds = &o
		}
		legs = append(legs, &legWork{in: in, matchID: id, market: m, odds: odds})
	}
	return legs, nil
}

// loadMatches fetches every match and refuses ones that can't be bet on.
func (s *TicketService) loadMatches(ctx context.Context, legs []*legWork) error {
	ids := make([]int64, len(legs))
	for i, l := range legs {
		ids[i] = l.matchID
	}
	rows, err := s.q.GetMatchesByIDs(ctx, ids)
	if err != nil {
		return err
	}
	byID := make(map[int64]dbsqlc.GetMatchesByIDsRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	now := s.now()
	for _, l := range legs {
		m, ok := byID[l.matchID]
		if !ok {
			return NotFound(fmt.Sprintf("Unknown match %q.", l.in.MatchID))
		}
		name := m.HomeName + " v " + m.AwayName
		switch {
		case m.Status == "postponed":
			return Invalid(name + " has been postponed. Remove it from your ticket.")
		case m.Status != "scheduled" || !m.Kickoff.After(now):
			return Invalid(name + " has already kicked off. Remove it from your ticket.")
		}
		l.match = m
	}
	return nil
}

// loadProfiles reads each side's home or away profile. A team without one (no
// results yet) gets an empty profile, which the engine marks low confidence.
func (s *TicketService) loadProfiles(ctx context.Context, legs []*legWork) error {
	var teams []teamSide
	for _, l := range legs {
		teams = append(teams,
			teamSide{l.match.HomeID, l.match.HomeName, engine.Home},
			teamSide{l.match.AwayID, l.match.AwayName, engine.Away})
	}
	profiles, err := loadProfiles(ctx, s.q, s.engine, teams)
	if err != nil {
		return err
	}
	for i, l := range legs {
		l.home, l.away = profiles[2*i], profiles[2*i+1]
	}
	return nil
}

// log writes the ticket log and builds the response legs (SRS addendum A6).
func (s *TicketService) log(ctx context.Context, q *dbsqlc.Queries, req AnalyseRequest, legs []*legWork) ([]AnalysedLeg, error) {
	existing, err := q.ListTicketLegs(ctx, req.TicketID)
	if err != nil {
		return nil, err
	}
	byMatch := map[int64]dbsqlc.AnalysedLeg{}
	var oldIDs []int64
	for _, e := range existing {
		byMatch[e.MatchID] = e
		oldIDs = append(oldIDs, e.ID)
	}
	suggested := map[int64]map[string]bool{} // leg id → markets suggested so far
	if len(oldIDs) > 0 {
		rows, err := q.ListLegSuggestions(ctx, oldIDs)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if suggested[r.AnalysedLegID] == nil {
				suggested[r.AnalysedLegID] = map[string]bool{}
			}
			suggested[r.AnalysedLegID][r.Market] = true
		}
	}

	out := make([]AnalysedLeg, 0, len(legs))
	keep := make([]int64, 0, len(legs))
	for _, l := range legs {
		market := string(l.market)
		var legID int64
		prev, seen := byMatch[l.matchID]
		switch {
		case !seen:
			legID, err = q.InsertTicketLeg(ctx, dbsqlc.InsertTicketLegParams{
				TicketID: req.TicketID, Origin: req.Origin, MatchID: l.matchID,
				OriginalMarket: market, OriginalOdds: l.odds,
				OriginalEstimate: l.est.Estimate, OriginalBand: string(l.est.Band),
			})
		case market != prev.OriginalMarket && suggested[prev.ID][market]:
			// An accepted suggestion: the final pick changes, the user's pick doesn't.
			legID = prev.ID
			err = q.UpdateTicketLeg(ctx, dbsqlc.UpdateTicketLegParams{
				ID: prev.ID, Origin: req.Origin,
				OriginalMarket: prev.OriginalMarket, OriginalOdds: prev.OriginalOdds,
				OriginalEstimate: prev.OriginalEstimate, OriginalBand: prev.OriginalBand,
				FinalMarket: market,
			})
		default:
			// The user's own pick: the same one again (or back after Undo), or a
			// new one, which drops the suggestions made for the old pick.
			legID = prev.ID
			if market != prev.OriginalMarket {
				if err = q.DeleteLegSuggestions(ctx, prev.ID); err != nil {
					return nil, err
				}
			}
			err = q.UpdateTicketLeg(ctx, dbsqlc.UpdateTicketLegParams{
				ID: prev.ID, Origin: req.Origin,
				OriginalMarket: market, OriginalOdds: l.odds,
				OriginalEstimate: l.est.Estimate, OriginalBand: string(l.est.Band),
				FinalMarket: market,
			})
		}
		if err != nil {
			return nil, err
		}
		keep = append(keep, legID)

		sugg := make([]Suggestion, 0, len(l.swaps))
		for _, sw := range l.swaps {
			id, err := q.UpsertSuggestion(ctx, dbsqlc.UpsertSuggestionParams{
				AnalysedLegID: legID, Market: string(sw.Market), Estimate: sw.Estimate, RiskBand: string(sw.Band),
			})
			if err != nil {
				return nil, err
			}
			sugg = append(sugg, Suggestion{
				ID: strconv.FormatInt(id, 10), Market: string(sw.Market),
				Estimate: sw.Estimate, RiskBand: string(sw.Band), Reason: sw.Reason,
			})
		}
		if err := q.SyncAcceptedSuggestions(ctx, legID); err != nil {
			return nil, err
		}
		out = append(out, s.responseLeg(legID, l, sugg))
	}
	if err := q.TakeLegsOffTicket(ctx, dbsqlc.TakeLegsOffTicketParams{TicketID: req.TicketID, Keep: keep}); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *TicketService) responseLeg(id int64, l *legWork, sugg []Suggestion) AnalysedLeg {
	m := l.match
	leg := AnalysedLeg{
		ID:            strconv.FormatInt(id, 10),
		MatchID:       strconv.FormatInt(m.ID, 10),
		Fixture:       fixture(m.ID, m.LeagueID, m.Kickoff, m.HomeID, m.HomeName, m.HomeForm, m.AwayID, m.AwayName, m.AwayForm),
		Market:        string(l.market),
		Odds:          l.odds,
		Estimate:      l.est.Estimate,
		RiskBand:      string(l.est.Band),
		Reason:        l.est.Reason,
		SampleSize:    l.est.SampleSize,
		WindowN:       s.engine.Thresholds().FormWindow,
		LowConfidence: l.est.LowConfidence,
		Suggestions:   sugg,
	}
	if l.odds != nil {
		implied := engine.Implied(*l.odds)
		value := s.engine.IsValue(l.est.Estimate, *l.odds)
		leg.ImpliedProb, leg.IsValue = &implied, &value
	}
	return leg
}

// combine works out the combined chance (FR-6) and the weakest leg (lowest
// estimate; the first one on a tie).
func (s *TicketService) combine(legs []AnalysedLeg) (CombinedProbability, *string) {
	estimates := make([]float64, len(legs))
	allOdds := true
	product := 1.0
	weakest := 0
	for i, l := range legs {
		estimates[i] = l.Estimate
		if l.Odds == nil {
			allOdds = false
		} else {
			product *= *l.Odds
		}
		if l.Estimate < legs[weakest].Estimate {
			weakest = i
		}
	}
	p := engine.Combined(estimates)
	c := CombinedProbability{Probability: p, OneIn: engine.OneIn(p)}
	if allOdds {
		c.Odds = &product
	}
	var id *string
	if len(legs) > 0 {
		id = &legs[weakest].ID
	}
	return c, id
}

// Accept records that the user took a suggestion (FR-5.3). It also counts as
// accepted when the ticket is next analysed with that market.
func (s *TicketService) Accept(ctx context.Context, suggestionID string) error {
	id, err := strconv.ParseInt(suggestionID, 10, 64)
	if err != nil || id <= 0 {
		return NotFound("Unknown suggestion.")
	}
	n, err := s.q.AcceptSuggestion(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return NotFound("Unknown suggestion.")
	}
	return nil
}

// ---------- profiles, shared with the wizard ----------

type teamSide struct {
	id    int64
	name  string
	venue engine.Venue
}

// loadProfiles returns one profile per side, in order, in one query.
func loadProfiles(ctx context.Context, q *dbsqlc.Queries, eng *engine.Engine, sides []teamSide) ([]engine.Profile, error) {
	ids := make([]int64, len(sides))
	for i, t := range sides {
		ids[i] = t.id
	}
	rows, err := q.ListProfilesForTeams(ctx, dbsqlc.ListProfilesForTeamsParams{
		TeamIds: ids, WindowN: int16(eng.Thresholds().FormWindow),
	})
	if err != nil {
		return nil, err
	}
	type key struct {
		id    int64
		venue string
	}
	byKey := make(map[key]dbsqlc.TeamProfile, len(rows))
	for _, r := range rows {
		byKey[key{r.TeamID, r.Venue}] = r
	}
	leagues, err := leagueProfilesFor(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	out := make([]engine.Profile, len(sides))
	for i, t := range sides {
		p := engine.Profile{Team: t.name, Venue: t.venue, League: leagues[t.id][t.venue]}
		if r, ok := byKey[key{t.id, string(t.venue)}]; ok {
			p.SampleSize = int(r.SampleSize)
			p.WinRate, p.DrawRate, p.LossRate = r.WinRate, r.DrawRate, r.LossRate
			p.GoalsForAvg, p.GoalsAgainstAvg = r.GfAvg, r.GaAvg
			p.ScoredRate, p.CleanSheetRate, p.BTTSRate = r.ScoredRate, r.CleanSheetRate, r.BttsRate
			p.Over15Rate, p.Over25Rate, p.Under35Rate = r.Over15Rate, r.Over25Rate, r.Under35Rate
		}
		out[i] = p
	}
	return out, nil
}

// leagueProfilesFor returns each team's league average at home and away, which
// estimates pull towards (prior_games). A league with no profile yet (before
// its first import) gives nil, and the team's rates are used as they are.
func leagueProfilesFor(ctx context.Context, q *dbsqlc.Queries, teamIDs []int64) (map[int64]map[engine.Venue]*engine.Profile, error) {
	rows, err := q.ListLeagueProfiles(ctx)
	if err != nil {
		return nil, err
	}
	byLeague := map[string]map[engine.Venue]*engine.Profile{}
	for _, r := range rows {
		if byLeague[r.LeagueID] == nil {
			byLeague[r.LeagueID] = map[engine.Venue]*engine.Profile{}
		}
		byLeague[r.LeagueID][engine.Venue(r.Venue)] = &engine.Profile{
			Team: r.LeagueID, Venue: engine.Venue(r.Venue), SampleSize: int(r.SampleSize),
			WinRate: r.WinRate, DrawRate: r.DrawRate, LossRate: r.LossRate,
			GoalsForAvg: r.GfAvg, GoalsAgainstAvg: r.GaAvg,
			ScoredRate: r.ScoredRate, CleanSheetRate: r.CleanSheetRate, BTTSRate: r.BttsRate,
			Over15Rate: r.Over15Rate, Over25Rate: r.Over25Rate, Under35Rate: r.Under35Rate,
		}
	}
	teams, err := q.ListTeamLeagues(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]map[engine.Venue]*engine.Profile, len(teams))
	for _, t := range teams {
		out[t.ID] = byLeague[t.LeagueID]
	}
	return out, nil
}
