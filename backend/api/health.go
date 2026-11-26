package api

import "github.com/gofiber/fiber/v3"

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// health answers Render's health checks. It says the process is up; it does not
// check the database, so a database blip doesn't make Render restart the API.
func (a *API) health(c fiber.Ctx) error {
	return c.JSON(healthResponse{Status: "ok", Version: a.version})
}
