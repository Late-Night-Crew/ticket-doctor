// Package saved serves V2 saved tickets (FR-9): save a snapshot, open it by
// its private link, list a browser's saved tickets, delete one.
package saved

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"ticketdoctor/api/apierr"
	"ticketdoctor/services"
)

type Handler struct{ saved *services.SavedService }

func New(saved *services.SavedService) *Handler { return &Handler{saved: saved} }

func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/saved", noStore)
	g.Post("", h.save)
	g.Get("", h.list)
	g.Get("/:id", h.get)
	g.Delete("/:id", h.delete)
}

// noStore: a saved ticket is private to whoever has the link, so keep it out
// of shared caches.
func noStore(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.Next()
}

type ticketRef struct {
	TicketID string `json:"ticket_id"`
}

func (h *Handler) save(c fiber.Ctx) error {
	var req ticketRef
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	t, err := h.saved.Save(c.Context(), req.TicketID)
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(t)
}

// list returns summaries for ?ids=a,b,c (the browser's "My tickets").
func (h *Handler) list(c fiber.Ctx) error {
	var ids []string
	for _, id := range strings.Split(c.Query("ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	list, err := h.saved.List(c.Context(), ids)
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(list)
}

func (h *Handler) get(c fiber.Ctx) error {
	t, err := h.saved.Get(c.Context(), c.Params("id"))
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(t)
}

func (h *Handler) delete(c fiber.Ctx) error {
	var req ticketRef
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	if err := h.saved.Delete(c.Context(), c.Params("id"), req.TicketID); err != nil {
		return apierr.From(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
