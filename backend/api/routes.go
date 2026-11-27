package api

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"ticketdoctor/api/leagues"
	"ticketdoctor/api/results"
	"ticketdoctor/api/tickets"
	"ticketdoctor/api/wizard"
	"ticketdoctor/pkg/problemdetail"
	"ticketdoctor/services"
)

// postsPerMinute caps POST requests per client IP. Analysis runs on an explicit
// tap, so a real user stays far below it.
const postsPerMinute = 30

func (a *API) routes() {
	api := a.app.Group("/api")
	api.Get("/health", a.health)

	d := a.deps
	if d.Pool == nil {
		return
	}
	api.Use(limiter.New(limiter.Config{
		Next:       func(c fiber.Ctx) bool { return c.Method() != fiber.MethodPost },
		Max:        postsPerMinute,
		Expiration: time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return problemdetail.TooManyRequests(c, "Too many requests. Please wait a minute and try again.")
		},
	}))

	leagues.New(services.NewFixtureService(d.Pool, d.Now)).Register(api)
	tickets.New(services.NewTicketService(d.Pool, d.Engine, d.Now)).Register(api)
	wizard.New(services.NewWizardService(d.Pool, d.Engine, d.Now)).Register(api)
	results.New(services.NewResultsService(d.Pool)).Register(api)
}
