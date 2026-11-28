package services

import (
	"context"
	"testing"
	"time"

	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
)

func TestBacktestHasNoHindsight(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ctx := context.Background()
	// Arsenal beat Chelsea 3-0 at home, then lose the next home game 0-1.
	first := w.match(t, "Arsenal", "Chelsea", testNow.Add(-14*24*time.Hour))
	second := w.match(t, "Arsenal", "Everton", testNow.Add(-7*24*time.Hour))
	setScore(t, w, first, 3, 0)
	setScore(t, w, second, 0, 1)
	s := NewCalibrationService(w.pool, engine.Default())

	// Both games have thin profiles, so by default nothing is rated.
	res, err := s.Calibration(ctx, CalibrationRequest{Source: SourceBacktest, Market: "HOME_WIN"})
	if err != nil || res.Matches != 0 || res.SkippedLowConfidence != 2 || res.N != 0 {
		t.Fatalf("default: %+v (%v)", res, err)
	}

	res, err = s.Calibration(ctx, CalibrationRequest{Source: SourceBacktest, Market: "HOME_WIN", IncludeLowConfidence: true})
	if err != nil || res.N != 2 {
		t.Fatalf("with thin profiles: %+v (%v)", res, err)
	}
	// Game 1: no history, so 0%, and it happened. Game 2: rated only from game 1
	// (Arsenal won 1/1 at home, Everton has no away games), and it didn't happen.
	if b := res.Bins[0]; b.N != 1 || b.Actual != 1 {
		t.Fatalf("first game bin: %+v", b)
	}
	var high Bin
	for _, b := range res.Bins[1:] {
		if b.N > 0 {
			high = b
		}
	}
	if high.N != 1 || high.Actual != 0 || high.Predicted < 0.45 || high.Predicted > 0.55 {
		t.Fatalf("second game: %+v, want one estimate near 50%% (Arsenal 1/1 home wins, Everton no data) that lost", high)
	}
	if len(res.Markets) != 1 || res.Markets[0].Market != "HOME_WIN" || res.Markets[0].N != 2 {
		t.Fatalf("markets: %+v", res.Markets)
	}
}

func TestCalibrationFromTickets(t *testing.T) {
	w := newWorld(t, dbtest.New(t))
	ctx := context.Background()
	ac := w.match(t, "Arsenal", "Chelsea", testNow.Add(5*time.Hour))
	first := analyse(t, newTickets(w), TicketLegInput{MatchID: ac, Market: "HOME_WIN"})
	s := NewCalibrationService(w.pool, engine.Default())

	if res, err := s.Calibration(ctx, CalibrationRequest{Source: SourceTickets}); err != nil || res.N != 0 {
		t.Fatalf("before the game: %+v (%v)", res, err)
	}
	setScore(t, w, ac, 2, 0)
	res, err := s.Calibration(ctx, CalibrationRequest{Source: SourceTickets, Kind: "picks"})
	if err != nil || res.N != 1 || res.Markets[0].Market != "HOME_WIN" || res.Markets[0].Actual != 1 {
		t.Fatalf("picks: %+v (%v)", res, err)
	}
	if res.Markets[0].Predicted != round4(first.Legs[0].Estimate) || len(res.Bands) != 1 || res.Bands[0].Band != string(first.Legs[0].RiskBand) {
		t.Fatalf("estimate shown %.4f %s, got %+v %+v", first.Legs[0].Estimate, first.Legs[0].RiskBand, res.Markets[0], res.Bands)
	}
	res, err = s.Calibration(ctx, CalibrationRequest{Source: SourceTickets, Kind: "suggestions"})
	if err != nil || res.N != len(first.Legs[0].Suggestions) {
		t.Fatalf("suggestions: %d, want %d (%v)", res.N, len(first.Legs[0].Suggestions), err)
	}
	if _, err := s.Calibration(ctx, CalibrationRequest{Source: "astrology"}); !isStatus(err, 422) {
		t.Fatalf("bad source: %v", err)
	}
}
