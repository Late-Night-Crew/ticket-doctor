package footballdataorg

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRealResponse(t *testing.T) {
	// A trimmed copy of a real /v4/competitions/PL/matches response, with three
	// statuses changed to cover them.
	f, err := os.Open("testdata/PL_matches.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d fixtures, want 4", len(got))
	}
	first := got[0]
	want := Fixture{
		ID:        560658,
		Kickoff:   time.Date(2026, 11, 27, 20, 0, 0, 0, time.UTC),
		Season:    "2627",
		APIStatus: "TIMED",
		Status:    Scheduled,
		Home:      Team{Name: "Nottingham Forest FC", ShortName: "Nottingham"},
		Away:      Team{Name: "Chelsea FC", ShortName: "Chelsea"},
	}
	if first != want {
		t.Fatalf("first fixture =\n %+v\nwant\n %+v", first, want)
	}
	for i, wantStatus := range []Status{Scheduled, Scheduled, Postponed, Skip} {
		if got[i].Status != wantStatus {
			t.Errorf("fixture %d (%s) status = %q, want %q", i, got[i].APIStatus, got[i].Status, wantStatus)
		}
	}
}

func TestParseRejectsSurprises(t *testing.T) {
	match := func(status, date, start, home string) string {
		return `{"matches":[{"id":1,"utcDate":"` + date + `","status":"` + status + `",
			"season":{"startDate":"` + start + `"},
			"homeTeam":{"name":` + home + `,"shortName":null},"awayTeam":{"name":"B","shortName":"B"}}]}`
	}
	for name, body := range map[string]string{
		"not json":       `<html>`,
		"unknown status": match("ABANDONED", "2026-11-28T15:00:00Z", "2026-08-21", `"A"`),
		"bad kick-off":   match("TIMED", "28/11/2026", "2026-08-21", `"A"`),
		"bad season":     match("TIMED", "2026-11-28T15:00:00Z", "", `"A"`),
		"missing team":   match("TIMED", "2026-11-28T15:00:00Z", "2026-08-21", `null`),
	} {
		if _, err := Parse(strings.NewReader(body)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	// A null short name is fine: matching falls back to the full name.
	got, err := Parse(strings.NewReader(match("TIMED", "2026-11-28T15:00:00Z", "2026-08-21", `"A FC"`)))
	if err != nil || got[0].Home != (Team{Name: "A FC"}) {
		t.Fatalf("null short name: %+v, %v", got, err)
	}
}

func testClient(url string) *Client {
	c := NewClient("secret-token")
	c.BaseURL = url
	c.RetryWait = time.Millisecond
	c.MaxRateLimitWait = 10 * time.Millisecond
	return c
}

func TestClientSendsTokenAndDates(t *testing.T) {
	body, _ := os.ReadFile("testdata/PL_matches.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "secret-token" {
			t.Errorf("token header = %q", r.Header.Get("X-Auth-Token"))
		}
		if r.URL.Path != "/competitions/PL/matches" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if q := r.URL.Query(); q.Get("dateFrom") != "2026-11-25" || q.Get("dateTo") != "2026-12-11" {
			t.Errorf("dates = %s", r.URL.RawQuery)
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	// Dates are sent as UTC days, whatever the caller's zone.
	jhb := time.FixedZone("SAST", 2*60*60)
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, jhb)
	to := time.Date(2026, 12, 11, 12, 0, 0, 0, jhb)
	got, err := testClient(srv.URL).Fixtures(context.Background(), "PL", from, to)
	if err != nil || len(got) != 4 {
		t.Fatalf("got %d fixtures, err %v", len(got), err)
	}
}

func TestClientErrors(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		c := testClient("http://unused.invalid")
		c.Token = ""
		if _, err := c.Fixtures(context.Background(), "PL", time.Now(), time.Now()); !errors.Is(err, ErrNoToken) {
			t.Fatalf("err = %v, want ErrNoToken", err)
		}
	})

	for name, tc := range map[string]struct {
		status    int
		wantCalls int32
		wantErr   string
	}{
		"bad key":      {http.StatusForbidden, 1, "API key rejected"},
		"not found":    {http.StatusNotFound, 1, "unexpected answer"},
		"server error": {http.StatusBadGateway, 3, "server error"},
		"rate limited": {http.StatusTooManyRequests, 3, "rate limit"},
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-RequestCounter-Reset", "0")
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()
			_, err := testClient(srv.URL).Fixtures(context.Background(), "PL", time.Now(), time.Now())
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
			if strings.Contains(err.Error(), "secret-token") {
				t.Fatal("the error message contains the token")
			}
			if calls.Load() != tc.wantCalls {
				t.Fatalf("calls = %d, want %d", calls.Load(), tc.wantCalls)
			}
		})
	}
}

func TestClientRetriesThenSucceeds(t *testing.T) {
	body, _ := os.ReadFile("testdata/PL_matches.json")
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	got, err := testClient(srv.URL).Fixtures(context.Background(), "PL", time.Now(), time.Now())
	if err != nil || len(got) != 4 || calls.Load() != 2 {
		t.Fatalf("got %d fixtures after %d calls, err %v", len(got), calls.Load(), err)
	}
}

func TestRateLimitWait(t *testing.T) {
	c := NewClient("x")
	for header, want := range map[string]time.Duration{
		"":    65 * time.Second, // missing: wait the cap
		"12":  13 * time.Second,
		"500": 65 * time.Second, // capped
		"abc": 65 * time.Second,
	} {
		h := http.Header{}
		if header != "" {
			h.Set("X-RequestCounter-Reset", header)
		}
		if got := c.rateLimitWait(h); got != want {
			t.Errorf("reset %q: wait %v, want %v", header, got, want)
		}
	}
}
