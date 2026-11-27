package engine

import (
	"sort"
	"strings"
	"time"
)

// Venue is where a team played: its home profile and away profile are separate.
type Venue string

const (
	Home Venue = "home"
	Away Venue = "away"
)

// Profile is a team's record over its last N matches at one venue (FR-2.2).
// Rates are fractions between 0 and 1. JSON names match the frontend mock.
type Profile struct {
	Team            string  `json:"team"`
	Venue           Venue   `json:"venue"`
	SampleSize      int     `json:"sample_size"`
	WinRate         float64 `json:"win_rate"`
	DrawRate        float64 `json:"draw_rate"`
	LossRate        float64 `json:"loss_rate"`
	GoalsForAvg     float64 `json:"gf_avg"`
	GoalsAgainstAvg float64 `json:"ga_avg"`
	ScoredRate      float64 `json:"scored_rate"`
	CleanSheetRate  float64 `json:"clean_sheet_rate"`
	BTTSRate        float64 `json:"btts_rate"`
	Over15Rate      float64 `json:"over15_rate"`
	Over25Rate      float64 `json:"over25_rate"`
	Under35Rate     float64 `json:"under35_rate"`
}

// Result is one finished match seen from one team's side.
type Result struct {
	Date         time.Time
	GoalsFor     int
	GoalsAgainst int
}

// BuildProfile computes a profile from a team's results at one venue, using the
// most recent FormWindow matches (FR-2.1). Fewer results give a smaller sample;
// an empty list gives a profile with sample size 0.
func (e *Engine) BuildProfile(team string, venue Venue, results []Result) Profile {
	recent := latest(results, e.t.FormWindow)
	p := Profile{Team: team, Venue: venue, SampleSize: len(recent)}
	if len(recent) == 0 {
		return p
	}

	var wins, draws, losses, scored, clean, btts, o15, o25, u35, gf, ga int
	for _, r := range recent {
		total := r.GoalsFor + r.GoalsAgainst
		switch {
		case r.GoalsFor > r.GoalsAgainst:
			wins++
		case r.GoalsFor == r.GoalsAgainst:
			draws++
		default:
			losses++
		}
		if r.GoalsFor > 0 {
			scored++
		}
		if r.GoalsAgainst == 0 {
			clean++
		}
		if r.GoalsFor > 0 && r.GoalsAgainst > 0 {
			btts++
		}
		if total >= 2 {
			o15++
		}
		if total >= 3 {
			o25++
		}
		if total <= 3 {
			u35++
		}
		gf += r.GoalsFor
		ga += r.GoalsAgainst
	}

	n := float64(len(recent))
	rate := func(k int) float64 { return float64(k) / n }
	p.WinRate, p.DrawRate, p.LossRate = rate(wins), rate(draws), rate(losses)
	p.GoalsForAvg, p.GoalsAgainstAvg = rate(gf), rate(ga)
	p.ScoredRate, p.CleanSheetRate, p.BTTSRate = rate(scored), rate(clean), rate(btts)
	p.Over15Rate, p.Over25Rate, p.Under35Rate = rate(o15), rate(o25), rate(u35)
	return p
}

// LowConfidence reports whether the profile has too few matches (FR-2.3).
func (e *Engine) LowConfidence(p Profile) bool {
	return p.SampleSize < e.t.LowConfidenceBelow
}

// Form returns the last-5 results across all venues, oldest first, e.g.
// "W-D-L-W-W" (FR-2.4). Fewer than 5 results give a shorter string.
func Form(results []Result) string {
	recent := latest(results, 5)
	parts := make([]string, len(recent))
	// latest returns newest first; the form string reads oldest to newest.
	for i, r := range recent {
		letter := "L"
		switch {
		case r.GoalsFor > r.GoalsAgainst:
			letter = "W"
		case r.GoalsFor == r.GoalsAgainst:
			letter = "D"
		}
		parts[len(recent)-1-i] = letter
	}
	return strings.Join(parts, "-")
}

// latest returns up to n results, newest first, without changing the input.
func latest(results []Result, n int) []Result {
	sorted := make([]Result, len(results))
	copy(sorted, results)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Date.After(sorted[j].Date) })
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}
