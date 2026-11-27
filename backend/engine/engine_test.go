package engine

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// Profiles matching the worked example in SRS 4.4.
var (
	teamA = Profile{Team: "Team A", Venue: Home, SampleSize: 10,
		WinRate: 0.4, DrawRate: 0.3, LossRate: 0.3, ScoredRate: 0.6, CleanSheetRate: 0.3,
		BTTSRate: 0.5, Over15Rate: 0.6, Over25Rate: 0.3, Under35Rate: 0.8}
	teamB = Profile{Team: "Team B", Venue: Away, SampleSize: 10,
		WinRate: 0.2, DrawRate: 0.3, LossRate: 0.5, ScoredRate: 0.8, CleanSheetRate: 0.2,
		BTTSRate: 0.5, Over15Rate: 0.6, Over25Rate: 0.3, Under35Rate: 0.8}
)

func mustEstimate(t *testing.T, e *Engine, m Market, home, away Profile) MarketEstimate {
	t.Helper()
	est, err := e.Estimate(m, home, away)
	if err != nil {
		t.Fatal(err)
	}
	return est
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// SRS 4.4: Team B away win is 25% (High); suggestions are Under 3.5 (80%) and Team B to score (75%).
func TestWorkedExample(t *testing.T) {
	e := Default()
	pick := mustEstimate(t, e, AwayWin, teamA, teamB)
	if !near(pick.Estimate, 0.25) || pick.Band != High {
		t.Fatalf("away win = %.4f %s, want 0.25 high", pick.Estimate, pick.Band)
	}
	for _, s := range []string{"Team B won 2 of last 10 away games", "Team A lost 3 of last 10 at home"} {
		if !strings.Contains(pick.Reason, s) {
			t.Errorf("reason %q missing %q", pick.Reason, s)
		}
	}
	if got := mustEstimate(t, e, AwayToScore, teamA, teamB).Estimate; !near(got, 0.75) {
		t.Errorf("Team B to score = %.4f, want 0.75", got)
	}
	if got := mustEstimate(t, e, Under35, teamA, teamB).Estimate; !near(got, 0.80) {
		t.Errorf("under 3.5 = %.4f, want 0.80", got)
	}
	var swaps []Market
	for _, s := range e.Swaps(pick, teamA, teamB) {
		swaps = append(swaps, s.Market)
	}
	if !slices.Equal(swaps, []Market{Under35, AwayToScore}) {
		t.Fatalf("swaps = %v, want [UNDER_3_5 AWAY_TO_SCORE]", swaps)
	}
}

func TestBands(t *testing.T) {
	e := Default()
	for _, tt := range []struct {
		est  float64
		want Band
	}{{0.65, Low}, {0.6499, Medium}, {0.45, Medium}, {0.4499, High}, {0.65 - 1e-12, Low}} {
		if got := e.BandFor(tt.est); got != tt.want {
			t.Errorf("BandFor(%v) = %s, want %s", tt.est, got, tt.want)
		}
	}
}

func TestLimitedDataRaisesBand(t *testing.T) {
	e := Default()
	thin := teamB
	thin.SampleSize = 4
	est := mustEstimate(t, e, Under35, teamA, thin)
	if !near(est.Estimate, 0.8) || est.Band != Medium || !est.LowConfidence || est.SampleSize != 4 {
		t.Fatalf("got %+v", est)
	}
	if !strings.HasSuffix(est.Reason, "(limited data)") {
		t.Fatalf("reason %q should end with (limited data)", est.Reason)
	}
}

func TestUnknownMarket(t *testing.T) {
	if _, err := Default().Estimate("CORNERS_OVER_9", teamA, teamB); err == nil {
		t.Fatal("expected an error for an unknown market")
	}
	if Market("CORNERS_OVER_9").Valid() || !Over25.Valid() {
		t.Fatal("Valid() wrong")
	}
}

func TestSwapRules(t *testing.T) {
	e := Default()
	if s := e.Swaps(mustEstimate(t, e, Under35, teamA, teamB), teamA, teamB); len(s) != 0 {
		t.Errorf("Low-risk leg got swaps: %v", s)
	}
	thin := teamB
	thin.SampleSize = 3
	if s := e.Swaps(mustEstimate(t, e, AwayWin, teamA, thin), teamA, thin); len(s) != 0 {
		t.Errorf("low-confidence match got swaps: %v", s)
	}
	pick := mustEstimate(t, e, AwayWin, teamA, teamB)
	for _, s := range e.Swaps(pick, teamA, teamB) {
		if Contradicts(s.Market, pick.Market) {
			t.Errorf("swap %s contradicts the pick %s", s.Market, pick.Market)
		}
		if s.Estimate+eps < pick.Estimate+0.10 {
			t.Errorf("swap %s (%.2f) is not 10 points better than %.2f", s.Market, s.Estimate, pick.Estimate)
		}
	}
}

func TestContradictsExamples(t *testing.T) {
	for _, tt := range []struct {
		a, b Market
		want bool
	}{
		{AwayWin, DC1X, true}, {Over25, Under25, true}, {Over25, Under35, false},
		{BTTSYes, Under15, true}, {AwayWin, Under35, false}, {HomeWin, AwayToScore, false},
	} {
		if got := Contradicts(tt.a, tt.b); got != tt.want {
			t.Errorf("Contradicts(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
		if Contradicts(tt.a, tt.b) != Contradicts(tt.b, tt.a) {
			t.Errorf("Contradicts not symmetric for %s, %s", tt.a, tt.b)
		}
	}
}

// Contradicts must be exactly "no scoreline lets both win". Checked against
// Settle for every pair of markets and every score from 0-0 to 9-9.
func TestContradictsMatchesEveryScoreline(t *testing.T) {
	for _, a := range Markets {
		for _, b := range Markets {
			if a == b {
				continue
			}
			bothCanWin := false
			for h := 0; h <= 9 && !bothCanWin; h++ {
				for w := 0; w <= 9; w++ {
					oa, _ := Settle(a, h, w)
					ob, _ := Settle(b, h, w)
					if oa == Won && ob == Won {
						bothCanWin = true
						break
					}
				}
			}
			if Contradicts(a, b) == bothCanWin {
				t.Errorf("Contradicts(%s, %s) = %v, but a scoreline where both win exists: %v", a, b, Contradicts(a, b), bothCanWin)
			}
		}
	}
}

func TestCombinedAndValue(t *testing.T) {
	e := Default()
	if got := Combined([]float64{0.7, 0.7, 0.7, 0.7, 0.7}); math.Abs(got-0.16807) > 1e-9 {
		t.Errorf("Combined = %v", got)
	}
	if got := Combined(nil); got != 1 {
		t.Errorf("Combined(nil) = %v, want 1", got)
	}
	if got := OneIn(0); got != 0 {
		t.Errorf("OneIn(0) = %d, want 0", got)
	}
	if got := OneIn(0.16807); got != 6 {
		t.Errorf("OneIn = %d, want 6 (about 1 in 6)", got)
	}
	if !e.IsValue(0.55, 2.0) || e.IsValue(0.54, 2.0) {
		t.Error("value should need the estimate to beat 1/odds by 5 points")
	}
	if !near(FairOdds(0.25), 4) || !near(Implied(4), 0.25) {
		t.Error("fair or implied odds wrong")
	}
}

func TestThresholds(t *testing.T) {
	if err := DefaultThresholds().Validate(); err != nil {
		t.Fatalf("defaults invalid: %v", err)
	}
	bad := DefaultThresholds()
	bad.MediumFrom, bad.LowFrom = 0.7, 0.6 // bands crossed
	bad.BoldFloor = 0.9
	if _, err := New(bad); err == nil || !strings.Contains(err.Error(), "bands") || !strings.Contains(err.Error(), "bold floor") {
		t.Fatalf("expected both problems reported, got %v", err)
	}
	custom := DefaultThresholds()
	custom.LowFrom = 0.8
	e, err := New(custom)
	if err != nil {
		t.Fatal(err)
	}
	if e.BandFor(0.75) != Medium {
		t.Error("custom threshold not applied")
	}
	if e.Thresholds().LowFrom != 0.8 {
		t.Error("Thresholds() should return the engine's thresholds")
	}
}
