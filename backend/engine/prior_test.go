package engine

import (
	"math"
	"strings"
	"testing"
)

func TestPriorPullsTheNumberButNotTheReason(t *testing.T) {
	league := &Profile{WinRate: 0.45, DrawRate: 0.25, LossRate: 0.30}
	home := Profile{Team: "Arsenal", Venue: Home, SampleSize: 10, WinRate: 1, League: league}
	away := Profile{Team: "Leeds", Venue: Away, SampleSize: 10, LossRate: 1, League: &Profile{WinRate: 0.30, LossRate: 0.45}}

	on := Default() // prior_games 10
	off, _ := New(func() Thresholds { t := DefaultThresholds(); t.PriorGames = 0; return t }())

	raw, _ := off.Estimate(HomeWin, home, away)
	pulled, _ := on.Estimate(HomeWin, home, away)
	if raw.Estimate != 0.95 { // 100% capped
		t.Fatalf("without the prior: %v", raw.Estimate)
	}
	// Arsenal (10/10 home wins) halfway to 45%, Leeds (10/10 away losses) halfway to 45%.
	if want := (0.725 + 0.725) / 2; math.Abs(pulled.Estimate-want) > 1e-12 {
		t.Fatalf("with the prior: %v, want %v", pulled.Estimate, want)
	}
	// The sentence still quotes the real record (addendum A7).
	if pulled.Reason != raw.Reason || !strings.Contains(pulled.Reason, "won 10 of last 10") {
		t.Fatalf("reason changed: %q vs %q", pulled.Reason, raw.Reason)
	}

	// No league profile (before the first import): nothing changes.
	home.League, away.League = nil, nil
	if e, _ := on.Estimate(HomeWin, home, away); e.Estimate != 0.95 {
		t.Fatalf("without a league profile: %v", e.Estimate)
	}
}
