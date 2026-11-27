package engine

import (
	"errors"
	"fmt"
)

// Thresholds are the tunable numbers behind every rule (NFR-6). Defaults are
// the SRS values; config.yaml's "rules:" section can override them.
type Thresholds struct {
	// FormWindow is N, how many recent home or away matches a profile uses (FR-2.1).
	FormWindow int
	// LowConfidenceBelow: profiles with fewer matches are low confidence (FR-2.3).
	LowConfidenceBelow int
	// LowFrom and MediumFrom are the risk band boundaries (4.2).
	LowFrom    float64
	MediumFrom float64
	// SwapMinGain is how much higher a swap's estimate must be (4.3, step 3).
	SwapMinGain float64
	// MaxSuggestions is how many swaps to offer per leg (4.3, step 5).
	MaxSuggestions int
	// ValueMargin: a leg is value when its estimate beats implied odds by this (4.5).
	ValueMargin float64
	// BoldFloor is the lowest estimate the wizard will ever use (4.6).
	BoldFloor float64
}

// DefaultThresholds returns the values in the SRS.
func DefaultThresholds() Thresholds {
	return Thresholds{
		FormWindow:         10,
		LowConfidenceBelow: 5,
		LowFrom:            0.65,
		MediumFrom:         0.45,
		SwapMinGain:        0.10,
		MaxSuggestions:     2,
		ValueMargin:        0.05,
		BoldFloor:          0.35,
	}
}

// Validate rejects thresholds that would make the rules meaningless.
func (t Thresholds) Validate() error {
	var errs []error
	check := func(ok bool, format string, args ...any) {
		if !ok {
			errs = append(errs, fmt.Errorf(format, args...))
		}
	}
	check(t.FormWindow >= 1, "form window must be at least 1, got %d", t.FormWindow)
	check(t.LowConfidenceBelow >= 1 && t.LowConfidenceBelow <= t.FormWindow,
		"low-confidence cut-off must be between 1 and the form window, got %d", t.LowConfidenceBelow)
	check(t.MediumFrom > 0 && t.MediumFrom < t.LowFrom && t.LowFrom < 1,
		"bands need 0 < medium_from < low_from < 1, got %.2f and %.2f", t.MediumFrom, t.LowFrom)
	check(t.SwapMinGain > 0 && t.SwapMinGain < 1, "swap minimum gain must be between 0 and 1, got %.2f", t.SwapMinGain)
	check(t.MaxSuggestions >= 1, "at least one swap suggestion, got %d", t.MaxSuggestions)
	check(t.ValueMargin >= 0 && t.ValueMargin < 1, "value margin must be between 0 and 1, got %.2f", t.ValueMargin)
	check(t.BoldFloor > 0 && t.BoldFloor < t.MediumFrom,
		"bold floor must be above 0 and below medium_from, got %.2f", t.BoldFloor)
	return errors.Join(errs...)
}
