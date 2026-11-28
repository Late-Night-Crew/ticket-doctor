package services

import (
	"context"
	"fmt"
	"math"
	"slices"

	"ticketdoctor/database/dbsqlc"
	"ticketdoctor/engine"
)

// Calibration answers "when we say 70%, does it happen about 70% of the
// time?", from two sources:
//
//   - backtest: every finished match in the database replayed in kick-off
//     order, each rated only from results before it (no hindsight), with the
//     same engine and thresholds as the live site, for every market;
//   - tickets: the estimates people were actually shown, for their own picks
//     or for our suggestions, once the games finished.
//
// It also scores the estimates against a naive baseline that always predicts
// the market's average rate (the Brier skill score): above 0 means the team
// stats add something, 0 or below means they don't. This is the evidence for
// deciding whether better data or a trained model is worth paying for.

type CalibrationSource string

const (
	SourceBacktest CalibrationSource = "backtest"
	SourceTickets  CalibrationSource = "tickets"
)

type CalibrationRequest struct {
	Source CalibrationSource
	League string // optional
	Market string // optional
	// Kind is "picks" (users' own) or "suggestions" (ours), for tickets only.
	Kind string
	// IncludeLowConfidence keeps estimates from profiles with too few games
	// (backtest only; the app marks those as limited data).
	IncludeLowConfidence bool
	// Prior (backtest only) overrides the prior_games rule for an experiment;
	// nil uses the rules in use now.
	Prior *float64
}

// Bin is one slice of the reliability chart: estimates in [From, To).
type Bin struct {
	From      float64 `json:"from"`
	To        float64 `json:"to"`
	N         int     `json:"n"`
	Predicted float64 `json:"predicted"` // mean estimate in the bin
	Actual    float64 `json:"actual"`    // share that won
}

type BandStat struct {
	Band      string  `json:"band"`
	N         int     `json:"n"`
	Predicted float64 `json:"predicted"`
	Actual    float64 `json:"actual"`
}

type MarketStat struct {
	Market    string  `json:"market"`
	N         int     `json:"n"`
	Predicted float64 `json:"predicted"`
	Actual    float64 `json:"actual"`
	Brier     float64 `json:"brier"`
	// BaseBrier is the Brier score of always predicting this market's actual rate.
	BaseBrier float64 `json:"base_brier"`
	// Skill is 1 − Brier/BaseBrier: above 0 beats the naive baseline.
	Skill float64 `json:"skill"`
}

type Calibration struct {
	Source CalibrationSource `json:"source"`
	N      int               `json:"n"`
	// Matches is how many games the backtest rated (0 for tickets).
	Matches int `json:"matches"`
	// SkippedLowConfidence counts backtest games left out for thin profiles.
	SkippedLowConfidence int          `json:"skipped_low_confidence"`
	Brier                float64      `json:"brier"`
	Bins                 []Bin        `json:"bins"`
	Bands                []BandStat   `json:"bands"`
	Markets              []MarketStat `json:"markets"`
}

type CalibrationService struct {
	q      *dbsqlc.Queries
	engine *engine.Engine
}

func NewCalibrationService(db dbsqlc.DBTX, eng *engine.Engine) *CalibrationService {
	return &CalibrationService{q: dbsqlc.New(db), engine: eng}
}

func (s *CalibrationService) Calibration(ctx context.Context, req CalibrationRequest) (*Calibration, error) {
	if req.Market != "" && !engine.Market(req.Market).Valid() {
		return nil, Invalid(fmt.Sprintf("Unknown market %q.", req.Market))
	}
	var league *string
	if req.League != "" {
		league = &req.League
	}
	acc := newCalAccumulator()
	out := &Calibration{Source: req.Source}

	switch req.Source {
	case SourceBacktest, "":
		out.Source = SourceBacktest
		rows, err := s.q.ListFinishedMatches(ctx, league)
		if err != nil {
			return nil, err
		}
		matches, skipped := s.backtest(rows, req, acc)
		out.Matches, out.SkippedLowConfidence = matches, skipped

	case SourceTickets:
		type row struct {
			market, band string
			estimate     float64
			home, away   *int16
		}
		var rows []row
		switch req.Kind {
		case "", "picks":
			got, err := s.q.ListSettledPickEstimates(ctx, league)
			if err != nil {
				return nil, err
			}
			for _, r := range got {
				rows = append(rows, row{r.Market, r.Band, r.Estimate, r.HomeGoals, r.AwayGoals})
			}
		case "suggestions":
			got, err := s.q.ListSettledSuggestionEstimates(ctx, league)
			if err != nil {
				return nil, err
			}
			for _, r := range got {
				rows = append(rows, row{r.Market, r.Band, r.Estimate, r.HomeGoals, r.AwayGoals})
			}
		default:
			return nil, Invalid("kind must be picks or suggestions.")
		}
		for _, r := range rows {
			if req.Market != "" && r.market != req.Market {
				continue
			}
			o, err := settle(r.market, r.home, r.away)
			if err != nil {
				return nil, err
			}
			acc.add(r.market, r.band, r.estimate, o == engine.Won)
		}

	default:
		return nil, Invalid("source must be backtest or tickets.")
	}

	acc.fill(out)
	return out, nil
}

// backtest rates every finished match from the games before it.
func (s *CalibrationService) backtest(rows []dbsqlc.ListFinishedMatchesRow, req CalibrationRequest, acc *calAccumulator) (matches, skipped int) {
	eng := s.engine
	if req.Prior != nil {
		t := eng.Thresholds()
		t.PriorGames = *req.Prior
		if e, err := engine.New(t); err == nil {
			eng = e
		}
	}
	window := eng.Thresholds().FormWindow
	home := map[int64][]engine.Result{} // each team's home results so far, oldest first
	away := map[int64][]engine.Result{}
	tail := func(r []engine.Result) []engine.Result { return r[max(0, len(r)-window):] }

	// League averages per venue from earlier games, as the importer builds them.
	leagueHome := map[string][]engine.Result{}
	leagueAway := map[string][]engine.Result{}
	ltail := func(r []engine.Result) []engine.Result { return r[max(0, len(r)-engine.LeagueWindow):] }

	markets := engine.Markets
	if req.Market != "" {
		markets = []engine.Market{engine.Market(req.Market)}
	}
	for _, r := range rows {
		if r.HomeGoals == nil || r.AwayGoals == nil {
			continue
		}
		hg, ag := int(*r.HomeGoals), int(*r.AwayGoals)
		hp := eng.BuildProfile("", engine.Home, tail(home[r.HomeTeamID]))
		ap := eng.BuildProfile("", engine.Away, tail(away[r.AwayTeamID]))
		lowConf := eng.LowConfidence(hp) || eng.LowConfidence(ap)
		if lh := leagueHome[r.LeagueID]; len(lh) > 0 {
			lhp := eng.BuildLeagueProfile(r.LeagueID, engine.Home, ltail(lh))
			lap := eng.BuildLeagueProfile(r.LeagueID, engine.Away, ltail(leagueAway[r.LeagueID]))
			hp.League, ap.League = &lhp, &lap
		}

		// Record this result only after rating the match: no hindsight.
		hr := engine.Result{Date: r.Kickoff, GoalsFor: hg, GoalsAgainst: ag}
		ar := engine.Result{Date: r.Kickoff, GoalsFor: ag, GoalsAgainst: hg}
		home[r.HomeTeamID] = append(home[r.HomeTeamID], hr)
		away[r.AwayTeamID] = append(away[r.AwayTeamID], ar)
		leagueHome[r.LeagueID] = append(leagueHome[r.LeagueID], hr)
		leagueAway[r.LeagueID] = append(leagueAway[r.LeagueID], ar)

		if !req.IncludeLowConfidence && lowConf {
			skipped++
			continue
		}
		matches++
		for _, m := range markets {
			e, err := eng.Estimate(m, hp, ap)
			if err != nil {
				continue
			}
			o, err := engine.Settle(m, hg, ag)
			if err != nil {
				continue
			}
			acc.add(string(m), string(e.Band), e.Estimate, o == engine.Won)
		}
	}
	return matches, skipped
}

// ---------- accumulating ----------

const calBins = 10

type calCell struct {
	n        int
	sumPred  float64
	wins     int
	sumSqErr float64 // Σ (estimate − outcome)²
}

func (c *calCell) add(est float64, won bool) {
	y := 0.0
	if won {
		y = 1
		c.wins++
	}
	c.n++
	c.sumPred += est
	c.sumSqErr += (est - y) * (est - y)
}

func (c *calCell) predicted() float64 { return ratio(c.sumPred, c.n) }
func (c *calCell) actual() float64    { return ratio(float64(c.wins), c.n) }

type calAccumulator struct {
	all     calCell
	bins    [calBins]calCell
	bands   map[string]*calCell
	markets map[string]*calCell
}

func newCalAccumulator() *calAccumulator {
	return &calAccumulator{bands: map[string]*calCell{}, markets: map[string]*calCell{}}
}

func (a *calAccumulator) add(market, band string, est float64, won bool) {
	a.all.add(est, won)
	a.bins[min(calBins-1, int(est*calBins))].add(est, won)
	for _, m := range []struct {
		cells map[string]*calCell
		key   string
	}{{a.bands, band}, {a.markets, market}} {
		if m.cells[m.key] == nil {
			m.cells[m.key] = &calCell{}
		}
		m.cells[m.key].add(est, won)
	}
}

func (a *calAccumulator) fill(out *Calibration) {
	out.N = a.all.n
	out.Brier = round4(ratio(a.all.sumSqErr, a.all.n))
	out.Bins = make([]Bin, 0, calBins)
	for i, b := range a.bins {
		out.Bins = append(out.Bins, Bin{From: float64(i) / calBins, To: float64(i+1) / calBins, N: b.n,
			Predicted: round4(b.predicted()), Actual: round4(b.actual())})
	}
	out.Bands = []BandStat{}
	for _, band := range []engine.Band{engine.Low, engine.Medium, engine.High} {
		if c := a.bands[string(band)]; c != nil {
			out.Bands = append(out.Bands, BandStat{Band: string(band), N: c.n, Predicted: round4(c.predicted()), Actual: round4(c.actual())})
		}
	}
	out.Markets = []MarketStat{}
	for _, m := range engine.Markets {
		c := a.markets[string(m)]
		if c == nil {
			continue
		}
		p := c.actual()
		base := p * (1 - p) // Brier of always predicting the market's own rate
		brier := ratio(c.sumSqErr, c.n)
		skill := 0.0
		if base > 0 {
			skill = 1 - brier/base
		}
		out.Markets = append(out.Markets, MarketStat{Market: string(m), N: c.n, Predicted: round4(c.predicted()), Actual: round4(p),
			Brier: round4(brier), BaseBrier: round4(base), Skill: round4(skill)})
	}
	slices.SortStableFunc(out.Markets, func(x, y MarketStat) int { return y.N - x.N })
}

func ratio(a float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return a / float64(n)
}

func round4(v float64) float64 { return math.Round(v*1e4) / 1e4 }
