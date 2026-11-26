// Package engine is the rules engine: pure functions, no database or HTTP.
//
//	profile.go     team home/away profiles from recent matches (FR-2)
//	estimate.go    leg probability per market (SRS 4.1, addendum A1)
//	bands.go       Low / Medium / High risk, low-confidence step-up (4.2)
//	swaps.go       swap suggestions, never contradicting the pick (4.3, addendum A2)
//	combined.go    combined probability, implied odds, value (4.5)
//	wizard.go      wizard ticket selection by band (4.6, addendum A4)
//
// Its tests must match frontend/src/mocks/scorer.test.ts, including the 4.4 worked example.
package engine
