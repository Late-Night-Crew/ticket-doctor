package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ticketdoctor/api/admin"
	"ticketdoctor/config"
	"ticketdoctor/db/dbtest"
	"ticketdoctor/engine"
	"ticketdoctor/services"
)

const testPassword = "correct horse battery"

// newAdminAPI returns an API with one admin account and two EPL matches: one
// kicked off two hours ago with no result yet, one still to come.
func newAdminAPI(t *testing.T) (a *API, played, upcoming string) {
	t.Helper()
	pool := dbtest.New(t)
	ctx := context.Background()
	var home, away int64
	for name, id := range map[string]*int64{"Arsenal": &home, "Chelsea": &away} {
		if err := pool.QueryRow(ctx, `INSERT INTO teams (league_id, name, name_key) VALUES ('epl', $1, $1) RETURNING id`, name).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	for kickoff, id := range map[time.Time]*string{now.Add(-2 * time.Hour): &played, now.Add(5 * time.Hour): &upcoming} {
		h, aw := home, away
		if id == &upcoming {
			h, aw = away, home
		}
		if err := pool.QueryRow(ctx, `INSERT INTO matches (league_id, season, kickoff, home_team_id, away_team_id, status, source)
			VALUES ('epl', '2627', $1, $2, $3, 'scheduled', 'test') RETURNING id::text`, kickoff, h, aw).Scan(id); err != nil {
			t.Fatal(err)
		}
	}
	clock := func() time.Time { return now }
	if _, err := services.NewAuthService(pool, nil, 0, clock).CreateAdmin(ctx, "Joseph", testPassword); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.Admin.SessionSecret = strings.Repeat("s", 32)
	a = New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), "test", Deps{Pool: pool, Engine: engine.Default(), Now: clock})
	return a, played, upcoming
}

// signIn logs in and returns the session cookie.
func signIn(t *testing.T, a *API, username, password string) (*http.Cookie, *http.Response) {
	t.Helper()
	res := call(t, a, "POST", "/api/admin/login", `{"username":"`+username+`","password":"`+password+`"}`, nil)
	for _, c := range res.Cookies() {
		if c.Name == admin.CookieName {
			return c, res
		}
	}
	return nil, res
}

func authed(t *testing.T, a *API, cookie *http.Cookie, method, path, body string, out any) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(cookie)
	res := do(t, a, req)
	if out != nil {
		decode(t, res, out)
	}
	return res
}

func TestAdminSignIn(t *testing.T) {
	a, _, _ := newAdminAPI(t)

	var p problem
	if res := call(t, a, "GET", "/api/admin/me", "", &p); res.StatusCode != 401 {
		t.Fatalf("me without a session = %d, want 401", res.StatusCode)
	}

	// Wrong password and unknown user get the same answer.
	for _, creds := range [][2]string{{"joseph", "wrong password!!"}, {"nobody", testPassword}} {
		cookie, res := signIn(t, a, creds[0], creds[1])
		decode(t, res, &p)
		if res.StatusCode != 401 || cookie != nil || p.Detail != "Wrong username or password." {
			t.Fatalf("sign in as %s: %d %q cookie=%v", creds[0], res.StatusCode, p.Detail, cookie)
		}
	}

	// Usernames are case-insensitive.
	cookie, res := signIn(t, a, " JOSEPH ", testPassword)
	if res.StatusCode != 200 || cookie == nil {
		t.Fatalf("sign in: %d", res.StatusCode)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api/admin" {
		t.Fatalf("cookie = %+v, want HttpOnly, SameSite=Strict, Path=/api/admin", cookie)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}

	var me services.Admin
	if res := authed(t, a, cookie, "GET", "/api/admin/me", "", &me); res.StatusCode != 200 || me.Username != "joseph" || me.Role != "admin" {
		t.Fatalf("me: %d %+v", res.StatusCode, me)
	}

	// A tampered token is refused, and the bad cookie is cleared.
	forged := *cookie
	forged.Value = cookie.Value[:len(cookie.Value)-2] + "xx"
	if res := authed(t, a, &forged, "GET", "/api/admin/me", "", nil); res.StatusCode != 401 || len(res.Cookies()) == 0 {
		t.Fatalf("tampered token: %d, cookies %v", res.StatusCode, res.Cookies())
	}

	if res := authed(t, a, cookie, "POST", "/api/admin/logout", "{}", nil); res.StatusCode != 204 || res.Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout: %d %v", res.StatusCode, res.Cookies())
	}
}

func TestAdminSessionEnds(t *testing.T) {
	a, _, _ := newAdminAPI(t)
	auth := services.NewAuthService(a.deps.Pool, nil, 0, a.deps.Now)
	ctx := context.Background()

	// Changing the password ends older sessions. Postgres stores microseconds
	// and the token seconds, so wait for the next second first.
	cookie, _ := signIn(t, a, "joseph", testPassword)
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
	if err := auth.SetPassword(ctx, "joseph", "another long password"); err != nil {
		t.Fatal(err)
	}
	if res := authed(t, a, cookie, "GET", "/api/admin/me", "", nil); res.StatusCode != 401 {
		t.Fatalf("after a password change: %d, want 401", res.StatusCode)
	}

	// Disabling an account ends its session and stops it signing in.
	cookie, _ = signIn(t, a, "joseph", "another long password")
	if err := auth.SetDisabled(ctx, "joseph", true); err != nil {
		t.Fatal(err)
	}
	if res := authed(t, a, cookie, "GET", "/api/admin/me", "", nil); res.StatusCode != 401 {
		t.Fatalf("disabled session: %d, want 401", res.StatusCode)
	}
	if c, res := signIn(t, a, "joseph", "another long password"); res.StatusCode != 401 || c != nil {
		t.Fatalf("disabled sign in: %d", res.StatusCode)
	}

	// A session from the past is expired.
	if err := auth.SetDisabled(ctx, "joseph", false); err != nil {
		t.Fatal(err)
	}
	cookie, _ = signIn(t, a, "joseph", "another long password")
	later := New(a.cfg, a.logger, "test", Deps{Pool: a.deps.Pool, Engine: a.deps.Engine, Now: func() time.Time { return now.Add(13 * time.Hour) }})
	if res := authed(t, later, cookie, "GET", "/api/admin/me", "", nil); res.StatusCode != 401 {
		t.Fatalf("13 hours later: %d, want 401", res.StatusCode)
	}
}

func TestAdminLoginLimit(t *testing.T) {
	a, _, _ := newAdminAPI(t)
	for i := range 10 {
		if _, res := signIn(t, a, "joseph", "wrong password!!"); res.StatusCode != 401 {
			t.Fatalf("attempt %d: %d", i+1, res.StatusCode)
		}
	}
	// The eleventh is refused even with the right password.
	if _, res := signIn(t, a, "joseph", testPassword); res.StatusCode != 429 {
		t.Fatalf("after 10 failures: %d, want 429", res.StatusCode)
	}
}

func TestAdminResultEntry(t *testing.T) {
	a, played, upcoming := newAdminAPI(t)
	cookie, _ := signIn(t, a, "joseph", testPassword)

	var awaiting []services.AdminMatch
	if res := authed(t, a, cookie, "GET", "/api/admin/matches?view=awaiting", "", &awaiting); res.StatusCode != 200 ||
		len(awaiting) != 1 || awaiting[0].ID != played || awaiting[0].Home != "Arsenal" {
		t.Fatalf("awaiting: %d %+v", res.StatusCode, awaiting)
	}

	var p problem
	if res := authed(t, a, cookie, "PUT", "/api/admin/matches/"+upcoming+"/result", `{"home_goals":1,"away_goals":0}`, &p); res.StatusCode != 422 {
		t.Fatalf("result for a game not yet played: %d %q", res.StatusCode, p.Detail)
	}
	for _, body := range []string{`{"home_goals":1}`, `{"home_goals":-1,"away_goals":0}`, `{"home_goals":21,"away_goals":0}`} {
		if res := authed(t, a, cookie, "PUT", "/api/admin/matches/"+played+"/result", body, nil); res.StatusCode != 422 {
			t.Fatalf("%s: %d, want 422", body, res.StatusCode)
		}
	}
	if res := authed(t, a, cookie, "PUT", "/api/admin/matches/999999/result", `{"home_goals":1,"away_goals":0}`, nil); res.StatusCode != 404 {
		t.Fatalf("unknown match: %d", res.StatusCode)
	}

	// Not JSON: refused before it reaches the handler (cross-site forms can't send JSON).
	req := httptest.NewRequest("PUT", "/api/admin/matches/"+played+"/result", strings.NewReader("home_goals=1&away_goals=0"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	if res := do(t, a, req); res.StatusCode != 415 {
		t.Fatalf("form post: %d, want 415", res.StatusCode)
	}

	// Enter 2-1, then correct it to 1-1: both entries are kept.
	for _, body := range []string{`{"home_goals":2,"away_goals":1}`, `{"home_goals":1,"away_goals":1}`} {
		if res := authed(t, a, cookie, "PUT", "/api/admin/matches/"+played+"/result", body, nil); res.StatusCode != 204 {
			t.Fatalf("enter %s: %d", body, res.StatusCode)
		}
	}
	var recent []services.AdminMatch
	authed(t, a, cookie, "GET", "/api/admin/matches?view=recent", "", &recent)
	if len(recent) != 1 || recent[0].Status != "finished" || recent[0].Source != "manual" || *recent[0].HomeGoals != 1 || *recent[0].AwayGoals != 1 {
		t.Fatalf("recent: %+v", recent)
	}
	var entries, prevHome int
	if err := a.deps.Pool.QueryRow(context.Background(),
		`SELECT count(*), max(previous_home) FROM result_entries WHERE match_id = $1`, played).Scan(&entries, &prevHome); err != nil || entries != 2 || prevHome != 2 {
		t.Fatalf("result_entries: %d entries, previous home %d (%v)", entries, prevHome, err)
	}
	if authed(t, a, cookie, "GET", "/api/admin/matches?view=awaiting", "", &awaiting); len(awaiting) != 0 {
		t.Fatalf("still awaiting: %+v", awaiting)
	}

	var dash services.Dashboard
	if res := authed(t, a, cookie, "GET", "/api/admin/dashboard", "", &dash); res.StatusCode != 200 ||
		dash.Matches.Finished != 1 || dash.Matches.Upcoming != 1 || dash.Matches.AwaitingResult != 0 {
		t.Fatalf("dashboard: %d %+v", res.StatusCode, dash)
	}

	var th []services.Threshold
	if res := authed(t, a, cookie, "GET", "/api/admin/thresholds", "", &th); res.StatusCode != 200 || len(th) != 9 ||
		th[len(th)-1].Key != "max_estimate" || th[len(th)-1].Value != 0.95 {
		t.Fatalf("thresholds: %d %+v", res.StatusCode, th)
	}
}

func TestAdminOffWithoutSecretInProduction(t *testing.T) {
	pool := dbtest.New(t)
	cfg := config.Defaults()
	cfg.Environment = config.Production
	a := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), "test", Deps{Pool: pool, Engine: engine.Default()})
	if res := call(t, a, "POST", "/api/admin/login", `{"username":"joseph","password":"`+testPassword+`"}`, nil); res.StatusCode != 503 {
		t.Fatalf("login without a secret in production: %d, want 503", res.StatusCode)
	}
}

func decode(t *testing.T, res *http.Response, out any) {
	t.Helper()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
