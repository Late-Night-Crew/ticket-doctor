// Package footballdataorg reads upcoming fixtures from the football-data.org
// API (v4), the fixtures source chosen in decision D4 (FR-1.2).
//
// Results still come from football-data.co.uk (package footballdata), which
// also has closing odds. This package only supplies kick-off times for games
// not yet played, and tells us when a game is postponed.
//
// The free tier allows 10 requests a minute with an API key (X-Auth-Token).
// An import makes one request per league.
package footballdataorg

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"ticketdoctor/pkg/footballdata"
)

// Status is what Ticket Doctor stores for a fixture.
type Status string

const (
	// Scheduled: the game has a kick-off time and can be picked.
	Scheduled Status = "scheduled"
	// Postponed: postponed, suspended or cancelled. Hidden from fixture lists
	// until the source gives it a new date.
	Postponed Status = "postponed"
	// Skip: in play or played. Results come from football-data.co.uk, so the
	// fixtures import leaves these alone.
	Skip Status = ""
)

// statuses maps the API's match statuses onto ours.
var statuses = map[string]Status{
	"SCHEDULED": Scheduled, // date known, time may not be
	"TIMED":     Scheduled, // date and time confirmed
	"POSTPONED": Postponed,
	"SUSPENDED": Postponed,
	"CANCELLED": Postponed,
	"IN_PLAY":   Skip,
	"PAUSED":    Skip,
	"FINISHED":  Skip,
	"AWARDED":   Skip,
}

// Fixture is one match from the API.
type Fixture struct {
	ID      int64
	Kickoff time.Time // UTC
	// Season is the football-data.co.uk season code, e.g. "2627", so fixtures
	// and results share one key for a match.
	Season string
	// APIStatus is the status as the API sent it; Status is ours.
	APIStatus string
	Status    Status
	Home      Team
	Away      Team
}

// Team carries both of the API's names: matching tries the full name first,
// then the short one.
type Team struct {
	Name      string
	ShortName string
}

// response is the part of GET /v4/competitions/{code}/matches we use.
type response struct {
	Matches []struct {
		ID      int64  `json:"id"`
		UTCDate string `json:"utcDate"`
		Status  string `json:"status"`
		Season  struct {
			StartDate string `json:"startDate"`
		} `json:"season"`
		HomeTeam apiTeam `json:"homeTeam"`
		AwayTeam apiTeam `json:"awayTeam"`
	} `json:"matches"`
}

type apiTeam struct {
	Name      *string `json:"name"`
	ShortName *string `json:"shortName"`
}

// Parse reads a matches response. A match with an unknown status, a missing
// team (cup draws not yet made) or a bad date is an error: the API is
// structured, so any of these means our assumptions are out of date.
func Parse(r io.Reader) ([]Fixture, error) {
	var res response
	if err := json.NewDecoder(r).Decode(&res); err != nil {
		return nil, fmt.Errorf("read matches: %w", err)
	}
	out := make([]Fixture, 0, len(res.Matches))
	for _, m := range res.Matches {
		status, ok := statuses[m.Status]
		if !ok {
			return nil, fmt.Errorf("match %d: unknown status %q", m.ID, m.Status)
		}
		kickoff, err := time.Parse(time.RFC3339, m.UTCDate)
		if err != nil {
			return nil, fmt.Errorf("match %d: kick-off %q: %w", m.ID, m.UTCDate, err)
		}
		start, err := time.Parse(time.DateOnly, m.Season.StartDate)
		if err != nil {
			return nil, fmt.Errorf("match %d: season start %q: %w", m.ID, m.Season.StartDate, err)
		}
		home, away := team(m.HomeTeam), team(m.AwayTeam)
		if home.Name == "" || away.Name == "" {
			return nil, fmt.Errorf("match %d: a team is missing", m.ID)
		}
		out = append(out, Fixture{
			ID:        m.ID,
			Kickoff:   kickoff.UTC(),
			Season:    footballdata.SeasonOf(start),
			APIStatus: m.Status,
			Status:    status,
			Home:      home,
			Away:      away,
		})
	}
	return out, nil
}

func team(t apiTeam) Team {
	var out Team
	if t.Name != nil {
		out.Name = *t.Name
	}
	if t.ShortName != nil {
		out.ShortName = *t.ShortName
	}
	return out
}
