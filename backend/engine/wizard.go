package engine

import "sort"

// Risk is the wizard's per-game risk level.
type Risk string

const (
	Safe     Risk = "safe"
	Balanced Risk = "balanced"
	Bold     Risk = "bold"
)

// Focus filters the wizard's markets (4.6.1).
type Focus string

const (
	FocusWins  Focus = "wins"
	FocusGoals Focus = "goals"
	FocusMix   Focus = "mix"
)

// MatchCandidates is every usable market for one match.
type MatchCandidates struct {
	MatchID    string           `json:"match_id"`
	Candidates []MarketEstimate `json:"candidates"`
}

// WizardLeg is one chosen leg.
type WizardLeg struct {
	MatchID string `json:"match_id"`
	MarketEstimate
}

// Candidates lists a match's markets for a focus, dropping any that rest on a
// low-confidence profile (4.6.1–4.6.2).
func (e *Engine) Candidates(matchID string, home, away Profile, focus Focus) MatchCandidates {
	mc := MatchCandidates{MatchID: matchID}
	for _, est := range e.EstimateAll(home, away) {
		if focus != FocusMix && string(est.Market.Group()) != string(focus) {
			continue
		}
		if est.LowConfidence {
			continue
		}
		mc.Candidates = append(mc.Candidates, est)
	}
	return mc
}

type slot struct {
	band  Band
	count int
	// fallback lists safer bands to use when this band runs short; never riskier.
	fallback []Band
}

// slotsFor decides which band each leg comes from (addendum A4), so Bold
// really means longer odds:
//
//	Safe      all Low
//	Balanced  half Medium (rounded down), the rest Low
//	Bold      half High (rounded down, never below BoldFloor), the rest Medium
func slotsFor(risk Risk, n int) []slot {
	half := n / 2
	switch risk {
	case Balanced:
		return []slot{{Medium, half, []Band{Low}}, {Low, n - half, nil}}
	case Bold:
		return []slot{{High, half, []Band{Medium, Low}}, {Medium, n - half, []Band{Low}}}
	default:
		return []slot{{Low, n, nil}}
	}
}

// BuildTicket picks n legs, one per match (4.6.3–4.6.6). Within a band the
// best-supported pick wins. If too few qualify it returns fewer legs; it never
// loosens the rules. Legs come back best first.
func (e *Engine) BuildTicket(matches []MatchCandidates, risk Risk, n int) []WizardLeg {
	used := map[string]bool{}
	var out []WizardLeg

	// take adds up to count legs from band and returns how many are still missing.
	take := func(band Band, count int) int {
		var pool []WizardLeg
		for _, m := range matches {
			if used[m.MatchID] {
				continue
			}
			if best, ok := e.bestInBand(m, band); ok {
				pool = append(pool, best)
			}
		}
		sort.SliceStable(pool, func(i, j int) bool { return pool[i].Estimate > pool[j].Estimate })
		if len(pool) > count {
			pool = pool[:count]
		}
		for _, l := range pool {
			used[l.MatchID] = true
			out = append(out, l)
		}
		return count - len(pool)
	}

	slots := slotsFor(risk, n)
	// First every slot from its own band, then any gaps from safer bands.
	missing := make([]int, len(slots))
	for i, s := range slots {
		missing[i] = take(s.band, s.count)
	}
	for i, s := range slots {
		for _, band := range s.fallback {
			if missing[i] <= 0 {
				break
			}
			missing[i] = take(band, missing[i])
		}
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Estimate > out[j].Estimate })
	return out
}

func (e *Engine) bestInBand(m MatchCandidates, band Band) (WizardLeg, bool) {
	var best *MarketEstimate
	for i := range m.Candidates {
		c := &m.Candidates[i]
		if c.Band != band || c.Estimate+eps < e.t.BoldFloor {
			continue
		}
		// Strictly greater keeps the first of equal estimates (stable, like the frontend).
		if best == nil || c.Estimate > best.Estimate {
			best = c
		}
	}
	if best == nil {
		return WizardLeg{}, false
	}
	return WizardLeg{MatchID: m.MatchID, MarketEstimate: *best}, true
}
