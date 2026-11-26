// Package api builds the Fiber app. api.go wires dependencies, middleware.go
// holds cross-cutting middleware, routes.go registers every endpoint. Handlers
// live in one sub-package per area (leagues, tickets, wizard, results, admin).
package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"ticketdoctor/config"
	"ticketdoctor/pkg/problemdetail"
)

type API struct {
	app     *fiber.App
	cfg     *config.Config
	logger  *slog.Logger
	version string
}

// New builds the app with middleware and routes registered, ready to Start.
func New(cfg *config.Config, logger *slog.Logger, version string) *API {
	a := &API{
		app:     fiber.New(fiberConfig(cfg, logger)),
		cfg:     cfg,
		logger:  logger,
		version: version,
	}
	a.middleware()
	a.routes()
	return a
}

func fiberConfig(cfg *config.Config, logger *slog.Logger) fiber.Config {
	return fiber.Config{
		AppName:      "ticketdoctor",
		ErrorHandler: problemdetail.ErrorHandler(logger),
		// A 15-leg ticket is a few KB; anything far larger is not a real request.
		BodyLimit:    64 * 1024,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		// Behind Render's proxy the client IP comes from X-Forwarded-For, but only
		// when the request really came through a trusted proxy (prevents spoofing).
		TrustProxy:       len(cfg.Server.TrustedProxies) > 0,
		TrustProxyConfig: fiber.TrustProxyConfig{Proxies: cfg.Server.TrustedProxies},
		ProxyHeader:      fiber.HeaderXForwardedFor,
	}
}

// App exposes the Fiber app, for tests (app.Test).
func (a *API) App() *fiber.App { return a.app }

// Start serves until ctx is cancelled (Ctrl+C, or SIGTERM from Render), then
// finishes in-flight requests before returning.
func (a *API) Start(ctx context.Context) error {
	return a.app.Listen(a.cfg.ListenAddress(), fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
	})
}
