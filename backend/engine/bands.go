package engine

// Band is a leg's risk rating (4.2).
type Band string

const (
	Low    Band = "low"
	Medium Band = "medium"
	High   Band = "high"
)

// eps absorbs floating-point noise at the boundaries, so 0.65 computed as
// 0.6499999999 still counts as Low (same as the frontend).
const eps = 1e-9

// BandFor rates an estimate: Low from 65%, Medium from 45%, else High (defaults).
func (e *Engine) BandFor(estimate float64) Band {
	switch {
	case estimate+eps >= e.t.LowFrom:
		return Low
	case estimate+eps >= e.t.MediumFrom:
		return Medium
	default:
		return High
	}
}

// raise moves a band one step riskier (used for low-confidence profiles).
func raise(b Band) Band {
	if b == Low {
		return Medium
	}
	return High
}
