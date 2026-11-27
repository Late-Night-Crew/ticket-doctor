package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	_ "time/tzdata" // the zone database is built in, so the Alpine image needs no tzdata package

	"github.com/jackc/pgx/v5"

	"ticketdoctor/database/dbsqlc"
)

// CAT is Central Africa Time (UTC+2, no daylight saving): the zone that
// decides "today" and "this weekend" for everyone (SRS addendum A5).
var CAT = func() *time.Location {
	if loc, err := time.LoadLocation("Africa/Blantyre"); err == nil {
		return loc
	}
	return time.FixedZone("CAT", 2*60*60)
}()

// Window is a fixture list's time range.
type Window string

const (
	Today    Window = "today"
	Weekend  Window = "weekend"
	Upcoming Window = "upcoming"
)

// ParseWindow reads a window; empty means upcoming.
func ParseWindow(s string) (Window, error) {
	switch w := Window(s); w {
	case "":
		return Upcoming, nil
	case Today, Weekend, Upcoming:
		return w, nil
	}
	return "", Invalid(fmt.Sprintf("when must be today, weekend or upcoming, not %q", s))
}

// WindowRange returns the kick-off range for a window, in CAT: from now until
// midnight for today; from now (or Saturday 00:00) until the end of Sunday for
// the weekend; from now with no end for upcoming. to is nil for no end.
func WindowRange(w Window, now time.Time) (from time.Time, to *time.Time) {
	local := now.In(CAT)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, CAT)
	day := func(n int) *time.Time { t := midnight.AddDate(0, 0, n); return &t }

	switch w {
	case Today:
		return now, day(1)
	case Weekend:
		switch wd := local.Weekday(); wd {
		case time.Saturday:
			return now, day(2)
		case time.Sunday:
			return now, day(1)
		default:
			toSaturday := int(time.Saturday - wd)
			return *day(toSaturday), day(toSaturday + 2)
		}
	}
	return now, nil
}

// League is a supported league (SRS 2.5).
type League struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

// Team is one side of a fixture, with its last-5 form, oldest first (FR-2.4).
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Form string `json:"form,omitempty"`
}

// Fixture is a match as the frontend shows it. IDs are strings in JSON.
type Fixture struct {
	ID       string    `json:"id"`
	LeagueID string    `json:"league_id"`
	Kickoff  time.Time `json:"kickoff"`
	Home     Team      `json:"home"`
	Away     Team      `json:"away"`
}

// FixtureService lists leagues and upcoming fixtures (FR-1.2, FR-3.1).
type FixtureService struct {
	q   *dbsqlc.Queries
	now func() time.Time
}

func NewFixtureService(db dbsqlc.DBTX, now func() time.Time) *FixtureService {
	return &FixtureService{q: dbsqlc.New(db), now: now}
}

func (s *FixtureService) Leagues(ctx context.Context) ([]League, error) {
	rows, err := s.q.ListLeagues(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]League, 0, len(rows))
	for _, l := range rows {
		out = append(out, League{ID: l.ID, Name: l.Name, Country: l.Country})
	}
	return out, nil
}

// Fixtures lists a league's games in a window, earliest first. An empty
// league means every league.
func (s *FixtureService) Fixtures(ctx context.Context, league string, w Window) ([]Fixture, error) {
	if league != "" {
		if _, err := s.q.GetLeague(ctx, league); errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFound(fmt.Sprintf("No league called %q.", league))
		} else if err != nil {
			return nil, err
		}
	}
	from, to := WindowRange(w, s.now())
	params := dbsqlc.ListFixturesParams{KickoffFrom: from, KickoffTo: to}
	if league != "" {
		params.LeagueID = &league
	}
	rows, err := s.q.ListFixtures(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]Fixture, 0, len(rows))
	for _, r := range rows {
		out = append(out, fixture(r.ID, r.LeagueID, r.Kickoff, r.HomeID, r.HomeName, r.HomeForm, r.AwayID, r.AwayName, r.AwayForm))
	}
	return out, nil
}

func fixture(id int64, league string, kickoff time.Time, homeID int64, homeName, homeForm string, awayID int64, awayName, awayForm string) Fixture {
	return Fixture{
		ID:       strconv.FormatInt(id, 10),
		LeagueID: league,
		Kickoff:  kickoff.UTC(),
		Home:     Team{ID: strconv.FormatInt(homeID, 10), Name: homeName, Form: homeForm},
		Away:     Team{ID: strconv.FormatInt(awayID, 10), Name: awayName, Form: awayForm},
	}
}
