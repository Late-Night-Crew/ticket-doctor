package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"ticketdoctor/config"
	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
	"ticketdoctor/services"
)

var now = time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC) // Friday, noon in CAT

// newDataAPI returns an API on the test database with one upcoming EPL match.
func newDataAPI(t *testing.T) (*API, string) {
	t.Helper()
	pool := dbtest.New(t)
	ctx := context.Background()
	var home, away, match int64
	for name, id := range map[string]*int64{"Arsenal": &home, "Chelsea": &away} {
		if err := pool.QueryRow(ctx, `INSERT INTO teams (league_id, name, name_key) VALUES ('epl', $1, $1) RETURNING id`, name).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `INSERT INTO matches (league_id, season, kickoff, home_team_id, away_team_id, status, source)
		VALUES ('epl', '2627', $1, $2, $3, 'scheduled', 'test') RETURNING id`, now.Add(5*time.Hour), home, away).Scan(&match); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	a := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), "test", Deps{Pool: pool, Engine: engine.Default(), Now: func() time.Time { return now }})
	return a, strconv.FormatInt(match, 10)
}

func call(t *testing.T, a *API, method, path, body string, out any) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res := do(t, a, req)
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
	return res
}

type problem struct {
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func TestDataEndpoints(t *testing.T) {
	a, match := newDataAPI(t)

	var leagues []services.League
	if res := call(t, a, "GET", "/api/leagues", "", &leagues); res.StatusCode != 200 || len(leagues) != 3 {
		t.Fatalf("leagues: %d %v", res.StatusCode, leagues)
	}

	var fixtures []map[string]any
	if res := call(t, a, "GET", "/api/leagues/epl/fixtures?when=today", "", &fixtures); res.StatusCode != 200 || len(fixtures) != 1 || fixtures[0]["id"] != match {
		t.Fatalf("fixtures: %d %v", res.StatusCode, fixtures)
	}
	// IDs are strings and the kick-off is ISO 8601, as the frontend expects.
	if home := fixtures[0]["home"].(map[string]any); home["name"] != "Arsenal" || fixtures[0]["kickoff"] != "2026-11-27T15:00:00Z" {
		t.Fatalf("fixture shape: %v", fixtures[0])
	}

	var analysis map[string]any
	body := `{"ticket_id":"abcdef12-3456","origin":"manual","legs":[{"match_id":"` + match + `","market":"HOME_WIN","odds":null}]}`
	if res := call(t, a, "POST", "/api/tickets/analyse", body, &analysis); res.StatusCode != 200 || analysis["ticket_id"] != "abcdef12-3456" {
		t.Fatalf("analyse: %d %v", res.StatusCode, analysis)
	}
	leg := analysis["legs"].([]any)[0].(map[string]any)
	if leg["low_confidence"] != true || leg["odds"] != nil || analysis["weakest_leg_id"] != leg["id"] {
		t.Fatalf("analysed leg: %v", leg)
	}

	var wiz services.WizardResponse
	if res := call(t, a, "POST", "/api/wizard", `{"path":"guided","risk":"safe","focus":"mix","legs":2,"when":"today","seed":0}`, &wiz); res.StatusCode != 200 || wiz.ShortfallReason == nil {
		t.Fatalf("wizard (teams have no profiles, so nothing qualifies): %d %+v", res.StatusCode, wiz)
	}

	var results map[string]any
	if res := call(t, a, "GET", "/api/results?league=epl", "", &results); res.StatusCode != 200 || results["updated_at"] != nil || len(results["rows"].([]any)) != 0 {
		t.Fatalf("results: %d %v", res.StatusCode, results)
	}
}

func TestDataEndpointErrorsAreProblemDetails(t *testing.T) {
	a, match := newDataAPI(t)
	cases := []struct {
		method, path, body string
		status             int
		detail             string
	}{
		{"GET", "/api/leagues/seriea/fixtures", "", 404, `No league called "seriea".`},
		{"GET", "/api/leagues/epl/fixtures?when=tomorrow", "", 422, "when must be"},
		{"POST", "/api/tickets/analyse", `{"ticket_id":`, 400, "isn't valid JSON"},
		{"POST", "/api/tickets/analyse", `{"ticket_id":"abcdef12-3456","origin":"manual","legs":[{"match_id":"` + match + `","market":"DC_12"}]}`, 422, `Unknown market "DC_12".`},
		{"POST", "/api/suggestions/999/accept", "", 404, "Unknown suggestion."},
		{"POST", "/api/wizard", `{"path":"guided","risk":"safe","focus":"mix","legs":2,"when":"upcoming"}`, 422, "when must be today or weekend."},
		{"GET", "/api/results?market=NOPE", "", 422, "Unknown market"},
	}
	for _, c := range cases {
		var p problem
		res := call(t, a, c.method, c.path, c.body, &p)
		if res.StatusCode != c.status || p.Status != c.status || !strings.Contains(p.Detail, c.detail) {
			t.Errorf("%s %s: %d %+v, want %d %q", c.method, c.path, res.StatusCode, p, c.status, c.detail)
		}
		if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
			t.Errorf("%s %s: content type %q", c.method, c.path, ct)
		}
	}
}

func TestPostsAreRateLimited(t *testing.T) {
	a, _ := newDataAPI(t)
	var last *http.Response
	for range postsPerMinute + 1 {
		last = call(t, a, "POST", "/api/suggestions/1/accept", "", nil)
	}
	var p problem
	if err := json.NewDecoder(last.Body).Decode(&p); err != nil || last.StatusCode != 429 || !strings.Contains(p.Detail, "Too many requests") {
		t.Fatalf("request %d: %d %+v", postsPerMinute+1, last.StatusCode, p)
	}
	// GETs are not limited.
	if res := call(t, a, "GET", "/api/leagues", "", nil); res.StatusCode != 200 {
		t.Fatalf("GET after the POST limit: %d", res.StatusCode)
	}
}

// With render.yaml's TD_TRUSTED_PROXIES, the limit applies per visitor
// (X-Forwarded-For), not to Render's proxy address that every request shares.
func TestRateLimitIsPerVisitorBehindTrustedProxies(t *testing.T) {
	a, _ := newDataAPI(t)
	cfg := config.Defaults()
	if err := cfg.ApplyEnvironment(func(k string) (string, bool) {
		if k == "TD_TRUSTED_PROXIES" {
			return "0.0.0.0/0,::/0", true // the value in render.yaml
		}
		return "", false
	}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("render.yaml's proxies rejected: %v", err)
	}
	a = New(cfg, a.logger, "test", a.deps)

	post := func(visitor string) int {
		req := httptest.NewRequest("POST", "/api/suggestions/1/accept", nil)
		req.Header.Set("X-Forwarded-For", visitor)
		return do(t, a, req).StatusCode
	}
	for range postsPerMinute {
		post("198.51.100.7")
	}
	if got := post("198.51.100.7"); got != 429 {
		t.Fatalf("visitor over the limit got %d, want 429", got)
	}
	if got := post("203.0.113.9"); got == 429 {
		t.Fatal("another visitor was limited too: the limit is keyed on the proxy, not the visitor")
	}
}
