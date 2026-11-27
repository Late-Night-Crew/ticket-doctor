// Package engine is the Ticket Doctor rules engine (SRS section 4 and the SRS
// addendum): pure functions with no database or HTTP, so every rule can be
// tested on its own.
//
//	markets.go     the 15 supported markets (FR-3.2)
//	thresholds.go  the tunable rule numbers (NFR-6)
//	profile.go     home/away team profiles from recent matches (FR-2)
//	estimate.go    leg probability per market (4.1, addendum A1)
//	bands.go       Low / Medium / High risk, low-confidence step-up (4.2)
//	swaps.go       swap suggestions that never contradict the pick (4.3, addendum A2)
//	combined.go    combined probability, implied probability, value (4.5)
//	wizard.go      wizard ticket selection by band (4.6, addendum A4)
//	settle.go      won/lost for a market given the final score (FR-7.2)
//
// The frontend's mock API (frontend/src/mocks/scorer.ts) implements the same
// rules. testdata/parity.json is generated from it, and the tests here must
// reproduce it exactly, so the mock and the real API can never drift apart.
package engine

import "fmt"

// Engine applies the rules with a given set of thresholds.
type Engine struct {
	t Thresholds
}

// New returns an engine using t, after checking the thresholds make sense.
func New(t Thresholds) (*Engine, error) {
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("rule thresholds: %w", err)
	}
	return &Engine{t: t}, nil
}

// Default returns an engine with the SRS thresholds.
func Default() *Engine {
	return &Engine{t: DefaultThresholds()}
}

// Thresholds returns the thresholds this engine uses (e.g. for the admin view).
func (e *Engine) Thresholds() Thresholds { return e.t }
