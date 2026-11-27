package engine

import "math"

// Combined is the chance every leg wins, treating legs as independent (4.5).
func Combined(estimates []float64) float64 {
	p := 1.0
	for _, e := range estimates {
		p *= e
	}
	return p
}

// OneIn turns a probability into "about 1 in N" (RG-2), N at least 1.
func OneIn(p float64) int {
	if p <= 0 {
		return 0
	}
	return max(1, int(math.Floor(1/p+0.5)))
}

// Implied is the bookmaker's implied probability: 1 ÷ decimal odds (1.3).
func Implied(odds float64) float64 { return 1 / odds }

// FairOdds is 1 ÷ the engine's estimate: the odds if the estimate were exact.
func FairOdds(estimate float64) float64 { return 1 / estimate }

// IsValue reports whether the estimate beats the implied probability by at
// least ValueMargin (4.5, FR-4.4).
func (e *Engine) IsValue(estimate, odds float64) bool {
	return estimate+eps >= Implied(odds)+e.t.ValueMargin
}
