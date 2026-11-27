package engine

// Market is a bet type for one match. Values match the frontend's MarketKey.
type Market string

const (
	HomeWin     Market = "HOME_WIN"
	Draw        Market = "DRAW"
	AwayWin     Market = "AWAY_WIN"
	DC1X        Market = "DC_1X"
	DCX2        Market = "DC_X2"
	BTTSYes     Market = "BTTS_YES"
	BTTSNo      Market = "BTTS_NO"
	Over15      Market = "OVER_1_5"
	Under15     Market = "UNDER_1_5"
	Over25      Market = "OVER_2_5"
	Under25     Market = "UNDER_2_5"
	Over35      Market = "OVER_3_5"
	Under35     Market = "UNDER_3_5"
	HomeToScore Market = "HOME_TO_SCORE"
	AwayToScore Market = "AWAY_TO_SCORE"
)

// Markets lists every supported market in canonical order. The order matters:
// when two markets have the same estimate, the earlier one wins (the same as
// MARKETS in frontend/src/config/markets.ts).
var Markets = []Market{
	HomeWin, Draw, AwayWin, DC1X, DCX2,
	BTTSYes, BTTSNo, Over15, Under15, Over25, Under25, Over35, Under35, HomeToScore, AwayToScore,
}

// Group is the wizard focus a market belongs to (4.6.1).
type Group string

const (
	GroupWins  Group = "wins"
	GroupGoals Group = "goals"
)

// Group returns the market's group: match results or goals.
func (m Market) Group() Group {
	switch m {
	case HomeWin, Draw, AwayWin, DC1X, DCX2:
		return GroupWins
	default:
		return GroupGoals
	}
}

// Valid reports whether m is a supported market.
func (m Market) Valid() bool {
	for _, k := range Markets {
		if k == m {
			return true
		}
	}
	return false
}
