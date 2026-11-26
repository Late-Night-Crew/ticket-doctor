// Package footballdata reads results from football-data.co.uk (FR-1.1): one CSV
// per league and season, e.g. https://football-data.co.uk/mmz4281/2627/E0.csv.
//
// The files are UTF-8 with a byte-order mark, use UK dates (dd/mm/yyyy, or
// dd/mm/yy in older seasons) and UK kick-off times, and end with blank rows.
package footballdata

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // embed the timezone database, so Europe/London works in any container
)

// Match is one row: a finished match with its score and, where present, the
// market-average decimal odds.
type Match struct {
	Division string // e.g. "E0"
	Kickoff  time.Time
	HomeTeam string
	AwayTeam string
	// HomeGoals and AwayGoals are the full-time score.
	HomeGoals int
	AwayGoals int
	// Odds are nil when the file has none for this match.
	OddsHome, OddsDraw, OddsAway *float64
	// Line is the row's line number in the file (header is line 1), for messages.
	Line int
}

// RowError describes a row that was skipped.
type RowError struct {
	Line   int
	Reason string
}

func (e RowError) Error() string { return fmt.Sprintf("line %d: %s", e.Line, e.Reason) }

// UK kick-off times are local London time (GMT or BST).
var london = mustLocation("Europe/London")

// When a row has no kick-off time (older seasons), assume a 15:00 kick-off.
const defaultKickoff = "15:00"

// Odds columns in order of preference: market average, older-season average, Bet365.
var oddsColumns = [][3]string{
	{"AvgH", "AvgD", "AvgA"},
	{"BbAvH", "BbAvD", "BbAvA"},
	{"B365H", "B365D", "B365A"},
}

// Parse reads a results CSV. Rows that can't be read are skipped and reported
// in the returned RowErrors; blank rows are ignored silently. A file without the
// required columns is an error.
func Parse(r io.Reader) ([]Match, []RowError, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // trailing rows are often shorter than the header
	cr.ReuseRecord = false

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, errors.New("empty file")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}
	col := map[string]int{}
	for i, name := range header {
		col[strings.TrimSpace(strings.TrimPrefix(name, "\uFEFF"))] = i
	}
	for _, required := range []string{"Div", "Date", "HomeTeam", "AwayTeam", "FTHG", "FTAG"} {
		if _, ok := col[required]; !ok {
			return nil, nil, fmt.Errorf("missing column %q (is this a football-data.co.uk results file?)", required)
		}
	}

	var matches []Match
	var skipped []RowError
	line := 1
	for {
		rec, err := cr.Read()
		line++
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			skipped = append(skipped, RowError{line, err.Error()})
			continue
		}
		get := func(name string) string {
			i, ok := col[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		if get("HomeTeam") == "" && get("AwayTeam") == "" {
			continue // blank trailing row
		}

		m, err := parseRow(get, line)
		if err != nil {
			skipped = append(skipped, RowError{line, err.Error()})
			continue
		}
		matches = append(matches, m)
	}
	return matches, skipped, nil
}

func parseRow(get func(string) string, line int) (Match, error) {
	m := Match{Division: get("Div"), HomeTeam: get("HomeTeam"), AwayTeam: get("AwayTeam"), Line: line}
	if m.HomeTeam == "" || m.AwayTeam == "" {
		return m, errors.New("missing team name")
	}
	if m.HomeTeam == m.AwayTeam {
		return m, fmt.Errorf("home and away team are both %q", m.HomeTeam)
	}

	kickoff, err := parseKickoff(get("Date"), get("Time"))
	if err != nil {
		return m, err
	}
	m.Kickoff = kickoff

	if m.HomeGoals, err = goals(get("FTHG")); err != nil {
		return m, fmt.Errorf("home goals: %w", err)
	}
	if m.AwayGoals, err = goals(get("FTAG")); err != nil {
		return m, fmt.Errorf("away goals: %w", err)
	}

	for _, cols := range oddsColumns {
		h, d, a := odds(get(cols[0])), odds(get(cols[1])), odds(get(cols[2]))
		if h != nil && d != nil && a != nil {
			m.OddsHome, m.OddsDraw, m.OddsAway = h, d, a
			break
		}
	}
	return m, nil
}

func parseKickoff(date, clock string) (time.Time, error) {
	if date == "" {
		return time.Time{}, errors.New("missing date")
	}
	if clock == "" {
		clock = defaultKickoff
	}
	for _, layout := range []string{"02/01/2006 15:04", "02/01/06 15:04"} {
		if t, err := time.ParseInLocation(layout, date+" "+clock, london); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unreadable date/time %q %q", date, clock)
}

func goals(s string) (int, error) {
	if s == "" {
		return 0, errors.New("no score (match not played?)")
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 30 {
		return 0, fmt.Errorf("unreadable score %q", s)
	}
	return n, nil
}

// odds returns nil for missing or impossible odds (decimal odds are above 1).
func odds(s string) *float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 1 || v > 1000 {
		return nil
	}
	return &v
}

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
