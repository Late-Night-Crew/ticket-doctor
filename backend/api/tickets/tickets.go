// Package tickets serves POST /api/tickets/analyse and
// POST /api/suggestions/:id/accept (FR-3 to FR-6, FR-7.1).
package tickets

import (
	"github.com/gofiber/fiber/v3"

	"ticketdoctor/api/apierr"
	"ticketdoctor/services"
)

type Handler struct{ tickets *services.TicketService }

func New(tickets *services.TicketService) *Handler { return &Handler{tickets: tickets} }

func (h *Handler) Register(r fiber.Router) {
	r.Post("/tickets/analyse", h.analyse)
	r.Post("/suggestions/:id/accept", h.accept)
}

func (h *Handler) analyse(c fiber.Ctx) error {
	var req services.AnalyseRequest
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.tickets.Analyse(c.Context(), req)
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(res)
}

func (h *Handler) accept(c fiber.Ctx) error {
	if err := h.tickets.Accept(c.Context(), c.Params("id")); err != nil {
		return apierr.From(err)
	}
	return c.JSON(fiber.Map{"ok": true})
}
