// Package apierr connects service errors to HTTP responses for the handler
// packages.
package apierr

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"

	"ticketdoctor/services"
)

// From turns a services.UserError into a *fiber.Error with its status and
// message, which the error handler sends as problem details. Other errors pass
// through unchanged and become a logged 500.
func From(err error) error {
	if ue, ok := services.AsUserError(err); ok {
		return fiber.NewError(ue.Status, ue.Msg)
	}
	return err
}

// DecodeJSON reads the request body, rejecting malformed JSON with a 400.
func DecodeJSON(c fiber.Ctx, out any) error {
	if err := json.Unmarshal(c.Body(), out); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "The request body isn't valid JSON.")
	}
	return nil
}
