package footballdata

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Seasons run August to May. A season code is the two-digit start and end
// years: 2026/27 is "2627".

var seasonPattern = regexp.MustCompile(`^(\d{2})(\d{2})$`)

// SeasonOf returns the season a date falls in. July counts as the next season
// (pre-season), so 1 July 2026 is "2627".
func SeasonOf(t time.Time) string {
	y := t.Year() % 100
	if t.Month() >= time.July {
		return fmt.Sprintf("%02d%02d", y, (y+1)%100)
	}
	return fmt.Sprintf("%02d%02d", (y+99)%100, y)
}

// PreviousSeason returns the season before, e.g. "2627" → "2526".
func PreviousSeason(season string) (string, error) {
	if err := ValidSeason(season); err != nil {
		return "", err
	}
	start, _ := strconv.Atoi(season[:2])
	return fmt.Sprintf("%02d%02d", (start+99)%100, start), nil
}

// ValidSeason checks a season code: four digits, second year one after the first.
func ValidSeason(season string) error {
	m := seasonPattern.FindStringSubmatch(season)
	if m == nil {
		return fmt.Errorf("season %q: want four digits like 2627", season)
	}
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[2])
	if (a+1)%100 != b {
		return fmt.Errorf("season %q: second year must follow the first (e.g. 2627)", season)
	}
	return nil
}
