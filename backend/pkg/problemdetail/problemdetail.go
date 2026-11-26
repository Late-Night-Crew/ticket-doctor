// Package problemdetail writes RFC 7807 error responses, the same convention as
// nsms and parrot:
//
//	{"type":"about:blank","title":"Not Found","status":404,"detail":"..."}
//
// The frontend shows "detail" to the user as is, so keep it plain and helpful.
package problemdetail

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

const ContentType = "application/problem+json"

type ProblemDetail struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Write sends a problem response with the given status and user-facing detail.
func Write(c fiber.Ctx, status int, detail string) error {
	return c.Status(status).JSON(ProblemDetail{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}, ContentType)
}

func BadRequest(c fiber.Ctx, detail string) error { return Write(c, fiber.StatusBadRequest, detail) }
func NotFound(c fiber.Ctx, detail string) error   { return Write(c, fiber.StatusNotFound, detail) }
func UnprocessableEntity(c fiber.Ctx, detail string) error {
	return Write(c, fiber.StatusUnprocessableEntity, detail)
}
func TooManyRequests(c fiber.Ctx, detail string) error {
	return Write(c, fiber.StatusTooManyRequests, detail)
}

// ServerError hides internals from the client; log the real error before calling it.
func ServerError(c fiber.Ctx, detail string) error {
	return Write(c, fiber.StatusInternalServerError, detail)
}

// ErrorHandler turns any error a handler returns into a problem response.
// *fiber.Error (e.g. unknown route, body too large) keeps its status and message;
// anything else is logged and answered with a generic 500.
func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return Write(c, fe.Code, fe.Message)
		}
		logger.Error("unhandled error", "error", err, "method", c.Method(), "path", c.Path())
		return ServerError(c, "Something went wrong on our side. Please try again.")
	}
}
