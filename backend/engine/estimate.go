package engine

import (
	"fmt"
	"math"
)

// MarketEstimate is the engine's view of one market in one match.
type MarketEstimate struct {
	Market   Market  `json:"market"`
	Estimate float64 `json:"estimate"`
	Band     Band    `json:"risk_band"`
	// Reason is one plain sentence with the numbers behind the estimate (FR-4.3).
	Reason string `json:"reason"`
	// SampleSize is the smaller of the two profiles' samples (NFR-5).
	SampleSize    int  `json:"sample_size"`
	LowConfidence bool `json:"low_confidence"`
}

// Estimate rates one market for a match from the home team's home profile and
// the away team's away profile (4.1). Each estimate averages the two teams'
// views of the same event. Markets the SRS doesn't define directly use
// complements (addendum A1): BTTS no, Under 1.5, Under 2.5, Over 3.5.
func (e *Engine) Estimate(m Market, home, away Profile) (MarketEstimate, error) {
	h, a := home, away
	hn, an := h.SampleSize, a.SampleSize

	homeWin := avg(h.WinRate, a.LossRate)
	awayWin := avg(a.WinRate, h.LossRate)
	btts := avg(h.BTTSRate, a.BTTSRate)
	over15 := avg(h.Over15Rate, a.Over15Rate)
	over25 := avg(h.Over25Rate, a.Over25Rate)
	under35 := avg(h.Under35Rate, a.Under35Rate)

	var est float64
	var reason string
	switch m {
	case HomeWin:
		est = homeWin
		reason = fmt.Sprintf("%s won %d of last %d home games; %s lost %d of last %d away.",
			h.Team, count(h.WinRate, hn), hn, a.Team, count(a.LossRate, an), an)
	case AwayWin:
		est = awayWin
		reason = fmt.Sprintf("%s won %d of last %d away games; %s lost %d of last %d at home.",
			a.Team, count(a.WinRate, an), an, h.Team, count(h.LossRate, hn), hn)
	case Draw:
		est = avg(h.DrawRate, a.DrawRate)
		reason = fmt.Sprintf("%s drew %d of last %d at home; %s drew %d of last %d away.",
			h.Team, count(h.DrawRate, hn), hn, a.Team, count(a.DrawRate, an), an)
	case DC1X:
		est = 1 - awayWin
		reason = fmt.Sprintf("%s won only %d of last %d away; %s lost %d of last %d at home.",
			a.Team, count(a.WinRate, an), an, h.Team, count(h.LossRate, hn), hn)
	case DCX2:
		est = 1 - homeWin
		reason = fmt.Sprintf("%s won %d of last %d at home; %s lost %d of last %d away.",
			h.Team, count(h.WinRate, hn), hn, a.Team, count(a.LossRate, an), an)
	case HomeToScore:
		est = avg(h.ScoredRate, 1-a.CleanSheetRate)
		reason = fmt.Sprintf("%s scored in %d of last %d home games; %s kept %d clean sheets in %d away.",
			h.Team, count(h.ScoredRate, hn), hn, a.Team, count(a.CleanSheetRate, an), an)
	case AwayToScore:
		est = avg(a.ScoredRate, 1-h.CleanSheetRate)
		reason = fmt.Sprintf("%s scored in %d of last %d away games; %s kept %d clean sheets in %d at home.",
			a.Team, count(a.ScoredRate, an), an, h.Team, count(h.CleanSheetRate, hn), hn)
	case BTTSYes:
		est = btts
		reason = fmt.Sprintf("Both teams scored in %d of %s's last %d home games and %d of %s's last %d away.",
			count(h.BTTSRate, hn), h.Team, hn, count(a.BTTSRate, an), a.Team, an)
	case BTTSNo:
		est = 1 - btts
		reason = fmt.Sprintf("At least one side failed to score in %d of %s's last %d home games and %d of %s's last %d away.",
			hn-count(h.BTTSRate, hn), h.Team, hn, an-count(a.BTTSRate, an), a.Team, an)
	case Over15:
		est = over15
		reason = goalsReason("2 or more goals", h.Over15Rate, a.Over15Rate, h, a)
	case Under15:
		est = 1 - over15
		reason = goalsReason("1 goal or none", 1-h.Over15Rate, 1-a.Over15Rate, h, a)
	case Over25:
		est = over25
		reason = goalsReason("3 or more goals", h.Over25Rate, a.Over25Rate, h, a)
	case Under25:
		est = 1 - over25
		reason = goalsReason("2 goals or fewer", 1-h.Over25Rate, 1-a.Over25Rate, h, a)
	case Under35:
		est = under35
		reason = goalsReason("3 goals or fewer", h.Under35Rate, a.Under35Rate, h, a)
	case Over35:
		est = 1 - under35
		reason = goalsReason("4 or more goals", 1-h.Under35Rate, 1-a.Under35Rate, h, a)
	default:
		return MarketEstimate{}, fmt.Errorf("unknown market %q", m)
	}

	sample := min(hn, an)
	low := sample < e.t.LowConfidenceBelow
	band := e.BandFor(est)
	if low {
		// Limited data: one band riskier, and say so (4.2).
		band = raise(band)
		reason += " (limited data)"
	}
	return MarketEstimate{Market: m, Estimate: est, Band: band, Reason: reason, SampleSize: sample, LowConfidence: low}, nil
}

// EstimateAll rates every market for a match, in canonical market order.
func (e *Engine) EstimateAll(home, away Profile) []MarketEstimate {
	out := make([]MarketEstimate, len(Markets))
	for i, m := range Markets {
		out[i], _ = e.Estimate(m, home, away) // every listed market is known
	}
	return out
}

func goalsReason(what string, hRate, aRate float64, h, a Profile) string {
	return fmt.Sprintf("%s in %d of %s's last %d home games and %d of %s's last %d away.",
		what, count(hRate, h.SampleSize), h.Team, h.SampleSize, count(aRate, a.SampleSize), a.Team, a.SampleSize)
}

func avg(a, b float64) float64 { return (a + b) / 2 }

// count turns a rate back into "k of n" for reasons (JavaScript Math.round for
// positive numbers: halves round up).
func count(rate float64, n int) int { return int(math.Floor(rate*float64(n) + 0.5)) }
