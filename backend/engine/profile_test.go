package engine

import (
	"testing"
	"time"
)

func day(d int) time.Time { return time.Date(2026, 8, d, 15, 0, 0, 0, time.UTC) }

func TestBuildProfile(t *testing.T) {
	e := Default()
	results := []Result{
		{day(1), 2, 0}, // W, clean sheet, over 1.5
		{day(2), 1, 1}, // D, btts, over 1.5
		{day(3), 0, 3}, // L, over 2.5
		{day(4), 3, 2}, // W, btts, over 3.5 (not under 3.5)
	}
	p := e.BuildProfile("Arsenal", Home, results)
	want := Profile{
		Team: "Arsenal", Venue: Home, SampleSize: 4,
		WinRate: 0.5, DrawRate: 0.25, LossRate: 0.25,
		GoalsForAvg: 1.5, GoalsAgainstAvg: 1.5,
		ScoredRate: 0.75, CleanSheetRate: 0.25, BTTSRate: 0.5,
		Over15Rate: 1, Over25Rate: 0.5, Under35Rate: 0.75,
	}
	if p != want {
		t.Fatalf("got  %+v\nwant %+v", p, want)
	}
	if !e.LowConfidence(p) {
		t.Error("4 matches should be low confidence (fewer than 5)")
	}
}

func TestBuildProfileUsesLatestWindow(t *testing.T) {
	e := Default()
	var results []Result
	// 5 old losses, then 10 recent wins, given out of order
	for d := 20; d > 10; d-- {
		results = append(results, Result{day(d), 1, 0})
	}
	for d := 1; d <= 5; d++ {
		results = append(results, Result{day(d), 0, 1})
	}
	p := e.BuildProfile("Leeds", Away, results)
	if p.SampleSize != 10 || p.WinRate != 1 {
		t.Fatalf("should use only the 10 latest matches (all wins), got sample %d win rate %.2f", p.SampleSize, p.WinRate)
	}
	if e.LowConfidence(p) {
		t.Error("10 matches is not low confidence")
	}
}

func TestBuildProfileEmpty(t *testing.T) {
	p := Default().BuildProfile("Newcomers", Home, nil)
	if p.SampleSize != 0 || p.WinRate != 0 {
		t.Fatalf("got %+v", p)
	}
}

func TestForm(t *testing.T) {
	results := []Result{
		{day(6), 2, 1}, // newest: W
		{day(1), 0, 0}, // oldest, outside the last 5
		{day(3), 0, 2}, // L
		{day(5), 1, 1}, // D
		{day(2), 3, 0}, // W
		{day(4), 1, 0}, // W
	}
	if got := Form(results); got != "W-L-W-D-W" {
		t.Fatalf("Form = %q, want W-L-W-D-W (oldest first)", got)
	}
	if got := Form(results[:2]); got != "D-W" {
		t.Fatalf("Form with 2 results = %q, want D-W", got)
	}
}
