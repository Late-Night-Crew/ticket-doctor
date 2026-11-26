package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"ticketdoctor/config"
	"ticketdoctor/pkg/problemdetail"
)

func newTestAPI(t *testing.T) *API {
	t.Helper()
	cfg := config.Defaults()
	cfg.Server.AllowedOrigins = []string{"https://ticketdoctor.app"}
	return New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), "test-1.2.3")
}

func do(t *testing.T, a *API, req *http.Request) *http.Response {
	t.Helper()
	res, err := a.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestHealth(t *testing.T) {
	res := do(t, newTestAPI(t), httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var body healthResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body != (healthResponse{Status: "ok", Version: "test-1.2.3"}) {
		t.Fatalf("body = %+v", body)
	}
	if res.Header.Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID header")
	}
}

func TestUnknownRouteIsProblemDetail(t *testing.T) {
	res := do(t, newTestAPI(t), httptest.NewRequest(http.MethodGet, "/api/nope", nil))
	if res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != problemdetail.ContentType {
		t.Fatalf("status %d, content type %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

func TestCORS(t *testing.T) {
	a := newTestAPI(t)
	preflight := func(origin string) string {
		req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		return do(t, a, req).Header.Get("Access-Control-Allow-Origin")
	}
	if got := preflight("https://ticketdoctor.app"); got != "https://ticketdoctor.app" {
		t.Fatalf("allowed origin got %q", got)
	}
	if got := preflight("https://evil.example"); got != "" {
		t.Fatalf("other origin should not be allowed, got %q", got)
	}
}
