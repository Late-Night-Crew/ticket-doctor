// Package leagues serves GET /api/leagues and GET /api/leagues/:id/fixtures.
package leagues

import (
	"github.com/gofiber/fiber/v3"

	"ticketdoctor/api/apierr"
	"ticketdoctor/services"
)

type Handler struct{ fixtures *services.FixtureService }

func New(fixtures *services.FixtureService) *Handler { return &Handler{fixtures: fixtures} }

func (h *Handler) Register(r fiber.Router) {
	r.Get("/leagues", h.list)
	r.Get("/leagues/:id/fixtures", h.listFixtures)
}

func (h *Handler) list(c fiber.Ctx) error {
	leagues, err := h.fixtures.Leagues(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(leagues)
}

// listFixtures lists a league's games: ?when=today|weekend|upcoming (default
// upcoming), with "today" and "this weekend" in Central Africa Time.
func (h *Handler) listFixtures(c fiber.Ctx) error {
	when, err := services.ParseWindow(c.Query("when"))
	if err != nil {
		return apierr.From(err)
	}
	fixtures, err := h.fixtures.Fixtures(c.Context(), c.Params("id"), when)
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(fixtures)
}
