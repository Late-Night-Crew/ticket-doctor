package engine

import "testing"

func TestSettle(t *testing.T) {
	tests := []struct {
		m          Market
		home, away int
		want       Outcome
	}{
		{HomeWin, 2, 1, Won}, {HomeWin, 1, 1, Lost},
		{Draw, 0, 0, Won}, {Draw, 1, 0, Lost},
		{AwayWin, 0, 2, Won}, {AwayWin, 2, 2, Lost},
		{DC1X, 1, 1, Won}, {DC1X, 0, 1, Lost},
		{DCX2, 1, 1, Won}, {DCX2, 1, 0, Lost},
		{BTTSYes, 1, 1, Won}, {BTTSYes, 2, 0, Lost},
		{BTTSNo, 2, 0, Won}, {BTTSNo, 1, 1, Lost},
		{Over15, 1, 1, Won}, {Over15, 1, 0, Lost},
		{Under15, 1, 0, Won}, {Under15, 1, 1, Lost},
		{Over25, 2, 1, Won}, {Over25, 1, 1, Lost},
		{Under25, 1, 1, Won}, {Under25, 2, 1, Lost},
		{Over35, 2, 2, Won}, {Over35, 2, 1, Lost},
		{Under35, 2, 1, Won}, {Under35, 2, 2, Lost},
		{HomeToScore, 1, 3, Won}, {HomeToScore, 0, 3, Lost},
		{AwayToScore, 3, 1, Won}, {AwayToScore, 3, 0, Lost},
	}
	for _, tt := range tests {
		got, err := Settle(tt.m, tt.home, tt.away)
		if err != nil || got != tt.want {
			t.Errorf("Settle(%s, %d-%d) = %s, %v; want %s", tt.m, tt.home, tt.away, got, err, tt.want)
		}
	}
}

func TestSettleErrors(t *testing.T) {
	if _, err := Settle(HomeWin, -1, 0); err == nil {
		t.Error("negative score should be an error")
	}
	if _, err := Settle("NOPE", 1, 0); err == nil {
		t.Error("unknown market should be an error")
	}
}

// Every market must settle without error: guards against adding a market to
// Markets but forgetting it in Settle.
func TestEveryMarketSettles(t *testing.T) {
	for _, m := range Markets {
		if _, err := Settle(m, 1, 0); err != nil {
			t.Errorf("%s: %v", m, err)
		}
	}
}
