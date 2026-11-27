package engine

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
)

// resultClash lists market pairs that can never both win on the match result
// or on both-teams-to-score. Goal lines are handled separately below.
var resultClash = map[Market][]Market{
	HomeWin: {AwayWin, Draw, DCX2},
	AwayWin: {HomeWin, Draw, DC1X},
	Draw:    {HomeWin, AwayWin},
	DC1X:    {AwayWin},
	DCX2:    {HomeWin},
	BTTSYes: {BTTSNo, Under15}, // both teams scoring means at least 2 goals
	BTTSNo:  {BTTSYes},
}

var goalLinePattern = regexp.MustCompile(`^(OVER|UNDER)_(\d)_5$`)

// goalLine returns ("OVER"|"UNDER", line) for goal markets, e.g. OVER_2_5 → OVER, 2.5.
func goalLine(m Market) (side string, line float64, ok bool) {
	match := goalLinePattern.FindStringSubmatch(string(m))
	if match == nil {
		return "", 0, false
	}
	n, _ := strconv.Atoi(match[2])
	return match[1], float64(n) + 0.5, true
}

// Contradicts reports whether two markets can never both win. Swaps skip these,
// so a swap never bets against the user's own pick (addendum A2).
func Contradicts(a, b Market) bool {
	if slices.Contains(resultClash[a], b) || slices.Contains(resultClash[b], a) {
		return true
	}
	sa, la, okA := goalLine(a)
	sb, lb, okB := goalLine(b)
	if okA && okB && sa != sb {
		over, under := la, lb
		if sa == "UNDER" {
			over, under = lb, la
		}
		// Over x needs at least x+0.5 goals; under y needs at most y-0.5.
		return over >= under
	}
	return false
}

// Swaps suggests better-supported markets in the same match for a Medium or
// High pick (4.3): every other market that doesn't contradict the pick, at
// least SwapMinGain higher, not resting on a low-confidence profile, best
// first, at most MaxSuggestions.
func (e *Engine) Swaps(pick MarketEstimate, home, away Profile) []MarketEstimate {
	if pick.Band == Low {
		return nil
	}
	var out []MarketEstimate
	for _, c := range e.EstimateAll(home, away) {
		if c.Market == pick.Market || Contradicts(c.Market, pick.Market) {
			continue
		}
		if c.Estimate+eps < pick.Estimate+e.t.SwapMinGain || c.LowConfidence {
			continue
		}
		out = append(out, c)
	}
	// Stable: equal estimates keep canonical market order, like the frontend.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Estimate > out[j].Estimate })
	if len(out) > e.t.MaxSuggestions {
		out = out[:e.t.MaxSuggestions]
	}
	return out
}
