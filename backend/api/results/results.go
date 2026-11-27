// Package results serves GET /api/results, the public track record (FR-7.3).
package results

import (
	"github.com/gofiber/fiber/v3"

	"ticketdoctor/api/apierr"
	"ticketdoctor/services"
)

type Handler struct{ results *services.ResultsService }

func New(results *services.ResultsService) *Handler { return &Handler{results: results} }

func (h *Handler) Register(r fiber.Router) { r.Get("/results", h.list) }

// list returns the track record, optionally ?league=epl&market=OVER_2_5.
func (h *Handler) list(c fiber.Ctx) error {
	res, err := h.results.Results(c.Context(), c.Query("league"), c.Query("market"))
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(res)
}
