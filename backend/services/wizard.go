package services

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
)

// Wizard limits (SRS FR-8 and addendum A3).
const (
	WizardMinLegs         = 2
	WizardMaxLegs         = 10
	WizardBeginnerMaxLegs = 4
)

type WizardRequest struct {
	Path  string  `json:"path"`
	Risk  string  `json:"risk"`
	Focus string  `json:"focus"`
	Legs  float64 `json:"legs"`
	When  string  `json:"when"`
	Seed  int64   `json:"seed"`
}

type WizardLeg struct {
	Fixture    Fixture `json:"fixture"`
	Market     string  `json:"market"`
	Estimate   float64 `json:"estimate"`
	RiskBand   string  `json:"risk_band"`
	Reason     string  `json:"reason"`
	SampleSize int     `json:"sample_size"`
	WindowN    int     `json:"window_n"`
	FairOdds   float64 `json:"fair_odds"`
}

type WizardResponse struct {
	Legs []WizardLeg `json:"legs"`
	// When is the window the ticket was built from: today or weekend. It
	// differs from the request only for "soonest".
	When            Window  `json:"when"`
	ShortfallReason *string `json:"shortfall_reason"`
}

// Soonest builds from today's games if they can fill the ticket, otherwise
// from this weekend's (the beginner path, which doesn't ask when; open
// question 9, decided 2026-11-27).
const Soonest Window = "soonest"

// WizardService builds tickets from the rules (FR-8, section 4.6).
type WizardService struct {
	q      *dbsqlc.Queries
	engine *engine.Engine
	now    func() time.Time
}

func NewWizardService(db dbsqlc.DBTX, eng *engine.Engine, now func() time.Time) *WizardService {
	return &WizardService{q: dbsqlc.New(db), engine: eng, now: now}
}

func (s *WizardService) Build(ctx context.Context, req WizardRequest) (*WizardResponse, error) {
	maxLegs := WizardMaxLegs
	switch req.Path {
	case "guided":
	case "beginner":
		maxLegs = WizardBeginnerMaxLegs
	default:
		return nil, Invalid("path must be beginner or guided.")
	}
	risk := engine.Risk(req.Risk)
	if risk != engine.Safe && risk != engine.Balanced && risk != engine.Bold {
		return nil, Invalid("risk must be safe, balanced or bold.")
	}
	focus := engine.Focus(req.Focus)
	if focus != engine.FocusWins && focus != engine.FocusGoals && focus != engine.FocusMix {
		return nil, Invalid("focus must be wins, goals or mix.")
	}
	when := Window(req.When)
	if when != Today && when != Weekend && when != Soonest {
		return nil, Invalid("when must be today, weekend or soonest.")
	}
	// Out-of-range numbers are clamped, as the frontend's mock does (addendum A3).
	target := int(math.Round(req.Legs))
	if math.IsNaN(req.Legs) || target < WizardMinLegs {
		target = WizardMinLegs
	}
	target = min(target, maxLegs)

	if when != Soonest {
		return s.build(ctx, when, risk, focus, target, req.Seed)
	}
	res, err := s.build(ctx, Today, risk, focus, target, req.Seed)
	if err != nil || len(res.Legs) >= target {
		return res, err
	}
	weekend, err := s.build(ctx, Weekend, risk, focus, target, req.Seed)
	if err != nil || len(weekend.Legs) <= len(res.Legs) {
		return res, err
	}
	return weekend, nil
}

// build makes a ticket from one window's fixtures.
func (s *WizardService) build(ctx context.Context, when Window, risk engine.Risk, focus engine.Focus, target int, seed int64) (*WizardResponse, error) {
	from, to := WindowRange(when, s.now())
	rows, err := s.q.ListFixtures(ctx, dbsqlc.ListFixturesParams{KickoffFrom: from, KickoffTo: to})
	if err != nil {
		return nil, err
	}
	sides := make([]teamSide, 0, 2*len(rows))
	for _, r := range rows {
		sides = append(sides, teamSide{r.HomeID, r.HomeName, engine.Home}, teamSide{r.AwayID, r.AwayName, engine.Away})
	}
	profiles, err := loadProfiles(ctx, s.q, s.engine, sides)
	if err != nil {
		return nil, err
	}
	fixtures := make(map[string]Fixture, len(rows))
	matches := make([]engine.MatchCandidates, 0, len(rows))
	for i, r := range rows {
		f := fixture(r.ID, r.LeagueID, r.Kickoff, r.HomeID, r.HomeName, r.HomeForm, r.AwayID, r.AwayName, r.AwayForm)
		fixtures[f.ID] = f
		matches = append(matches, s.engine.Candidates(f.ID, profiles[2*i], profiles[2*i+1], focus))
	}

	if seed > 0 {
		// Regenerate (FR-8.9): leave out a random third of the games, so a
		// different ticket appears, unless that leaves too few.
		rnd := rand.New(rand.NewPCG(uint64(seed), 0x7d))
		kept := make([]engine.MatchCandidates, 0, len(matches))
		for _, m := range matches {
			if rnd.Float64() > 0.33 {
				kept = append(kept, m)
			}
		}
		if len(kept) >= target {
			matches = kept
		}
	}

	picked := s.engine.BuildTicket(matches, risk, target)
	res := &WizardResponse{Legs: make([]WizardLeg, 0, len(picked)), When: when}
	for _, p := range picked {
		res.Legs = append(res.Legs, WizardLeg{
			Fixture:    fixtures[p.MatchID],
			Market:     string(p.Market),
			Estimate:   p.Estimate,
			RiskBand:   string(p.Band),
			Reason:     p.Reason,
			SampleSize: p.SampleSize,
			WindowN:    s.engine.Thresholds().FormWindow,
			FairOdds:   engine.FairOdds(p.Estimate),
		})
	}
	if len(picked) < target {
		day := "this weekend"
		if when == Today {
			day = "today"
		}
		msg := fmt.Sprintf("Only %d of the %d games you asked for met the %s rules for %s, so the ticket is shorter. We never loosen the rules to fill a ticket.",
			len(picked), target, risk, day)
		res.ShortfallReason = &msg
	}
	return res, nil
}
