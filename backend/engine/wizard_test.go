package engine

import (
	"fmt"
	"slices"
	"testing"
)

// match offers one pick per band: Low 80%, Medium 55%, High 40%, plus a 30%
// pick below the Bold floor. bump makes matches differ slightly.
func match(id string, bump float64) MatchCandidates {
	return MatchCandidates{MatchID: id, Candidates: []MarketEstimate{
		{Market: Under35, Estimate: 0.8 + bump, Band: Low, SampleSize: 10},
		{Market: BTTSYes, Estimate: 0.55 + bump, Band: Medium, SampleSize: 10},
		{Market: AwayWin, Estimate: 0.4 + bump, Band: High, SampleSize: 10},
		{Market: Draw, Estimate: 0.3, Band: High, SampleSize: 10},
	}}
}

func many(n int) []MatchCandidates {
	out := make([]MatchCandidates, n)
	for i := range out {
		out[i] = match(fmt.Sprintf("m%d", i), float64(i)*0.001)
	}
	return out
}

func bands(legs []WizardLeg) []Band {
	out := make([]Band, len(legs))
	for i, l := range legs {
		out[i] = l.Band
	}
	slices.Sort(out)
	return out
}

func chance(legs []WizardLeg) float64 {
	p := 1.0
	for _, l := range legs {
		p *= l.Estimate
	}
	return p
}

func TestWizardRiskLevels(t *testing.T) {
	e := Default()
	tests := []struct {
		risk Risk
		n    int
		want []Band
	}{
		{Safe, 5, []Band{Low, Low, Low, Low, Low}},
		{Balanced, 4, []Band{Low, Low, Medium, Medium}},
		{Balanced, 3, []Band{Low, Low, Medium}},
		{Bold, 4, []Band{High, High, Medium, Medium}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%d", tt.risk, tt.n), func(t *testing.T) {
			got := bands(e.BuildTicket(many(6), tt.risk, tt.n))
			want := slices.Clone(tt.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("bands = %v, want %v", got, want)
			}
		})
	}
}

func TestWizardBoldNeverBelowFloor(t *testing.T) {
	for _, l := range Default().BuildTicket(many(6), Bold, 6) {
		if l.Estimate < 0.35 {
			t.Fatalf("leg %s at %.2f is below the 35%% floor", l.Market, l.Estimate)
		}
	}
}

func TestWizardOneLegPerMatch(t *testing.T) {
	legs := Default().BuildTicket(many(10), Bold, 10)
	seen := map[string]bool{}
	for _, l := range legs {
		if seen[l.MatchID] {
			t.Fatalf("match %s used twice", l.MatchID)
		}
		seen[l.MatchID] = true
	}
}

// The trade-off from the wizard discussion: 2 Bold games pay more but win less
// often than 5 Safe games.
func TestWizardTwoBoldVersusFiveSafe(t *testing.T) {
	e := Default()
	bold2 := chance(e.BuildTicket(many(6), Bold, 2))
	safe5 := chance(e.BuildTicket(many(6), Safe, 5))
	if !(1/bold2 > 1/safe5) || !(bold2 < safe5) {
		t.Fatalf("2 Bold chance %.3f (odds %.2f) vs 5 Safe chance %.3f (odds %.2f)", bold2, 1/bold2, safe5, 1/safe5)
	}
}

func TestWizardFallsBackOnlyToSaferBands(t *testing.T) {
	e := Default()
	only := func(b Band) []MatchCandidates {
		ms := many(4)
		for i := range ms {
			ms[i].Candidates = slices.DeleteFunc(ms[i].Candidates, func(c MarketEstimate) bool { return c.Band != b })
		}
		return ms
	}
	if got := bands(e.BuildTicket(only(Low), Bold, 4)); !slices.Equal(got, []Band{Low, Low, Low, Low}) {
		t.Errorf("Bold with only Low picks = %v, want four Low", got)
	}
	if got := e.BuildTicket(only(High), Safe, 3); len(got) != 0 {
		t.Errorf("Safe must never use High picks, got %v", got)
	}
	if got := e.BuildTicket(only(High), Balanced, 3); len(got) != 0 {
		t.Errorf("Balanced must never use High picks, got %v", got)
	}
}

func TestWizardShortTicketRatherThanLooserRules(t *testing.T) {
	if got := Default().BuildTicket(many(2), Safe, 5); len(got) != 2 {
		t.Fatalf("got %d legs, want 2", len(got))
	}
}

func TestCandidatesFilterFocusAndLowConfidence(t *testing.T) {
	e := Default()
	wins := e.Candidates("m", teamA, teamB, FocusWins)
	for _, c := range wins.Candidates {
		if c.Market.Group() != GroupWins {
			t.Errorf("wins focus included %s", c.Market)
		}
	}
	if len(wins.Candidates) != 5 || len(e.Candidates("m", teamA, teamB, FocusMix).Candidates) != 15 {
		t.Errorf("unexpected candidate counts")
	}
	thin := teamB
	thin.SampleSize = 2
	if got := e.Candidates("m", teamA, thin, FocusMix).Candidates; len(got) != 0 {
		t.Errorf("low-confidence match should give no candidates, got %d", len(got))
	}
}
