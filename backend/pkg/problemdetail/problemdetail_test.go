package problemdetail

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	app.Get("/teapot", func(c fiber.Ctx) error { return fiber.NewError(fiber.StatusTeapot, "short and stout") })
	app.Get("/boom", func(c fiber.Ctx) error { return errors.New("db password is hunter2") })

	tests := []struct {
		path       string
		wantStatus int
		wantDetail string
	}{
		{"/teapot", 418, "short and stout"},
		{"/boom", 500, "Something went wrong on our side. Please try again."},
		{"/missing", 404, "Not Found"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			res, err := app.Test(httptest.NewRequest("GET", tt.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}
			if ct := res.Header.Get("Content-Type"); ct != ContentType {
				t.Fatalf("content type = %q", ct)
			}
			var p ProblemDetail
			if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
				t.Fatal(err)
			}
			if p.Status != tt.wantStatus || p.Detail != tt.wantDetail {
				t.Fatalf("problem = %+v", p)
			}
		})
	}
}
