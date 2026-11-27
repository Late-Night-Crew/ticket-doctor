package api

import (
	"crypto/rand"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"ticketdoctor/api/admin"
	"ticketdoctor/api/leagues"
	"ticketdoctor/api/results"
	"ticketdoctor/api/tickets"
	"ticketdoctor/api/wizard"
	"ticketdoctor/logging/loggedmodule"
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

	if secret := a.sessionSecret(); secret != nil {
		auth := services.NewAuthService(d.Pool, secret, time.Duration(a.cfg.Admin.SessionHours)*time.Hour, d.Now)
		adm := services.NewAdminService(d.Pool, d.Engine.Thresholds(), d.Now)
		admin.New(auth, adm, a.cfg.IsProduction()).Register(api)
	} else {
		admin.Disabled(api)
	}
}

// sessionSecret returns the key that signs admin sessions. Without one,
// development makes a random key (sessions end when the server restarts) and
// production turns admin sign-in off: a made-up key would sign people out on
// every deploy and hide the missing setting.
func (a *API) sessionSecret() []byte {
	if s := a.cfg.Admin.SessionSecret; s != "" {
		return []byte(s)
	}
	if a.cfg.IsProduction() {
		a.logger.Warn("admin sign-in is off: set TD_ADMIN_SESSION_SECRET", loggedmodule.Attr(loggedmodule.Server))
		return nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return key
}
