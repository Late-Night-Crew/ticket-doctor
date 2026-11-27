package engine

import "fmt"

// Outcome is a settled leg's result (FR-7.2).
type Outcome string

const (
	Won  Outcome = "won"
	Lost Outcome = "lost"
)

// Settle says whether a market won, given the full-time score.
func Settle(m Market, homeGoals, awayGoals int) (Outcome, error) {
	if homeGoals < 0 || awayGoals < 0 {
		return "", fmt.Errorf("invalid score %d-%d", homeGoals, awayGoals)
	}
	total := homeGoals + awayGoals
	var won bool
	switch m {
	case HomeWin:
		won = homeGoals > awayGoals
	case Draw:
		won = homeGoals == awayGoals
	case AwayWin:
		won = awayGoals > homeGoals
	case DC1X:
		won = homeGoals >= awayGoals
	case DCX2:
		won = awayGoals >= homeGoals
	case BTTSYes:
		won = homeGoals > 0 && awayGoals > 0
	case BTTSNo:
		won = homeGoals == 0 || awayGoals == 0
	case Over15:
		won = total >= 2
	case Under15:
		won = total <= 1
	case Over25:
		won = total >= 3
	case Under25:
		won = total <= 2
	case Over35:
		won = total >= 4
	case Under35:
		won = total <= 3
	case HomeToScore:
		won = homeGoals > 0
	case AwayToScore:
		won = awayGoals > 0
	default:
		return "", fmt.Errorf("unknown market %q", m)
	}
	if won {
		return Won, nil
	}
	return Lost, nil
}
