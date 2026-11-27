package api

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"ticketdoctor/logging/loggedmodule"
)

func (a *API) middleware() {
	// A panic in a handler becomes a 500 problem response instead of killing the server.
	a.app.Use(recover.New())
	// Every request gets an X-Request-ID, also written to the log line.
	a.app.Use(requestid.New())
	a.app.Use(a.requestLogger())
	// Only the frontend origins in config may call the API from a browser.
	a.app.Use(cors.New(cors.Config{
		AllowOrigins: a.cfg.Server.AllowedOrigins,
		AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodOptions},
		AllowHeaders: []string{fiber.HeaderContentType},
		MaxAge:       600,
	}))
}

// requestLogger writes one line per request. Health checks (Render polls them
// constantly) are logged at debug level only.
func (a *API) requestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// The error handler runs after middleware, so work out the status here.
		status := c.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			var fe *fiber.Error
			if errors.As(err, &fe) {
				status = fe.Code
			}
		}

		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case c.Path() == "/api/health":
			level = slog.LevelDebug
		}
		a.logger.Log(c.Context(), level, "request",
			loggedmodule.Attr(loggedmodule.API),
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestid.FromContext(c),
			"ip", c.IP(),
		)
		return err
	}
}
