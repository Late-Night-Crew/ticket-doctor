package engine

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// golden is testdata/parity.json, generated from the frontend's mock scorer
// (frontend/src/mocks/parity.test.ts). The Go engine must reproduce it exactly.
type golden struct {
	Matches []struct {
		MatchID   string              `json:"match_id"`
		Home      Profile             `json:"home"`
		Away      Profile             `json:"away"`
		Estimates []MarketEstimate    `json:"estimates"`
		Swaps     map[Market][]Market `json:"swaps"`
	} `json:"matches"`
	Wizard []struct {
		Focus  Focus `json:"focus"`
		Risk   Risk  `json:"risk"`
		Legs   int   `json:"legs"`
		Ticket []struct {
			MatchID string `json:"match_id"`
			Market  Market `json:"market"`
		} `json:"ticket"`
	} `json:"wizard"`
}

func loadGolden(t *testing.T) golden {
	t.Helper()
	raw, err := os.ReadFile("testdata/parity.json")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	var g golden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("parse golden file: %v", err)
	}
	if len(g.Matches) == 0 || len(g.Wizard) == 0 {
		t.Fatal("golden file is empty")
	}
	return g
}

func TestParityEstimatesAndSwaps(t *testing.T) {
	g := loadGolden(t)
	e := Default()
	for _, m := range g.Matches {
		got := e.EstimateAll(m.Home, m.Away)
		if len(got) != len(m.Estimates) {
			t.Fatalf("%s: %d estimates, golden has %d", m.MatchID, len(got), len(m.Estimates))
		}
		for i, want := range m.Estimates {
			have := got[i]
			if have.Market != want.Market || have.Band != want.Band || have.Reason != want.Reason ||
				have.SampleSize != want.SampleSize || have.LowConfidence != want.LowConfidence ||
				math.Abs(have.Estimate-want.Estimate) > 1e-12 {
				t.Errorf("%s %s:\n got  %+v\n want %+v", m.MatchID, want.Market, have, want)
			}

			var swaps []Market
			for _, s := range e.Swaps(have, m.Home, m.Away) {
				swaps = append(swaps, s.Market)
			}
			if !equalMarkets(swaps, m.Swaps[want.Market]) {
				t.Errorf("%s swaps for %s: got %v, want %v", m.MatchID, want.Market, swaps, m.Swaps[want.Market])
			}
		}
	}
}

func TestParityWizard(t *testing.T) {
	g := loadGolden(t)
	e := Default()
	for _, w := range g.Wizard {
		var matches []MatchCandidates
		for _, m := range g.Matches {
			matches = append(matches, e.Candidates(m.MatchID, m.Home, m.Away, w.Focus))
		}
		got := e.BuildTicket(matches, w.Risk, w.Legs)
		if len(got) != len(w.Ticket) {
			t.Errorf("%s/%s/%d: %d legs, want %d", w.Focus, w.Risk, w.Legs, len(got), len(w.Ticket))
			continue
		}
		for i, want := range w.Ticket {
			if got[i].MatchID != want.MatchID || got[i].Market != want.Market {
				t.Errorf("%s/%s/%d leg %d: got %s %s, want %s %s",
					w.Focus, w.Risk, w.Legs, i, got[i].MatchID, got[i].Market, want.MatchID, want.Market)
			}
		}
	}
}

func equalMarkets(a, b []Market) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
