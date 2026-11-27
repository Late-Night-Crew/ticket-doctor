// Package wizard serves POST /api/wizard (FR-8, SRS addendum A3/A4).
package wizard

import (
	"github.com/gofiber/fiber/v3"

	"ticketdoctor/api/apierr"
	"ticketdoctor/services"
)

type Handler struct{ wizard *services.WizardService }

func New(wizard *services.WizardService) *Handler { return &Handler{wizard: wizard} }

func (h *Handler) Register(r fiber.Router) { r.Post("/wizard", h.build) }

func (h *Handler) build(c fiber.Ctx) error {
	var req services.WizardRequest
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	res, err := h.wizard.Build(c.Context(), req)
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(res)
}
