package admin

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"ticketdoctor/api/apierr"
	"ticketdoctor/pkg/problemdetail"
	"ticketdoctor/services"
)

// CookieName holds the admin session token. It is HttpOnly (page scripts
// can't read it), SameSite=Strict (other sites can't make the browser send
// it) and limited to /api/admin.
const CookieName = "td_admin"

const cookiePath = "/api/admin"

// loginAttempts is how many failed sign-ins one address gets per window.
const (
	loginAttempts = 10
	loginWindow   = 15 * time.Minute
)

type Handler struct {
	auth  *services.AuthService
	admin *services.AdminService
	// secure marks the cookie HTTPS-only (always, except local development).
	secure bool
}

func New(auth *services.AuthService, admin *services.AdminService, secure bool) *Handler {
	return &Handler{auth: auth, admin: admin, secure: secure}
}

// Register adds the admin routes under r (which is /api).
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/admin", noStore, jsonOnly)
	// Handlers run in the order given: the limiter must come before login.
	g.Post("/login", limiter.New(limiter.Config{
		Max:                    loginAttempts,
		Expiration:             loginWindow,
		SkipSuccessfulRequests: true,
		LimitReached: func(c fiber.Ctx) error {
			return problemdetail.TooManyRequests(c, "Too many sign-in attempts. Please wait 15 minutes and try again.")
		},
	}), h.login)
	g.Post("/logout", h.logout)

	signedIn := g.Group("", h.requireAdmin)
	signedIn.Get("/me", h.me)
	signedIn.Get("/dashboard", h.dashboard)
	signedIn.Get("/matches", h.matches)
	signedIn.Put("/matches/:id/result", h.enterResult)
	signedIn.Get("/thresholds", h.thresholds)
}

// Disabled answers every admin route when no session secret is configured in
// production, rather than signing sessions with a made-up key.
func Disabled(r fiber.Router) {
	r.All("/admin/*", func(c fiber.Ctx) error {
		return problemdetail.Write(c, fiber.StatusServiceUnavailable, "Admin sign-in isn't set up on this server.")
	})
}

// noStore keeps admin responses out of every cache.
func noStore(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Next()
}

// jsonOnly refuses changes not sent as JSON. A form on another site can't send
// JSON without the browser asking first (CORS), so together with the
// SameSite cookie this stops cross-site request forgery.
func jsonOnly(c fiber.Ctx) error {
	if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead && !c.Is("json") {
		return problemdetail.Write(c, fiber.StatusUnsupportedMediaType, "Send the request as JSON.")
	}
	return c.Next()
}

func (h *Handler) requireAdmin(c fiber.Ctx) error {
	a, err := h.auth.Verify(c.Context(), c.Cookies(CookieName))
	if err != nil {
		if _, ok := services.AsUserError(err); ok {
			h.clearCookie(c)
		}
		return apierr.From(err)
	}
	c.Locals(adminKey{}, a)
	return c.Next()
}

type adminKey struct{}

func current(c fiber.Ctx) *services.Admin {
	a, _ := c.Locals(adminKey{}).(*services.Admin)
	return a
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionResponse struct {
	Admin     services.Admin `json:"admin"`
	ExpiresAt time.Time      `json:"expires_at"`
}

func (h *Handler) login(c fiber.Ctx) error {
	var req loginRequest
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	s, err := h.auth.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return apierr.From(err)
	}
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    s.Token,
		Path:     cookiePath,
		Expires:  s.ExpiresAt,
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
	return c.JSON(sessionResponse{Admin: s.Admin, ExpiresAt: s.ExpiresAt})
}

func (h *Handler) logout(c fiber.Ctx) error {
	h.clearCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) clearCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Path:     cookiePath,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

func (h *Handler) me(c fiber.Ctx) error { return c.JSON(current(c)) }

func (h *Handler) dashboard(c fiber.Ctx) error {
	d, err := h.admin.Dashboard(c.Context())
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(d)
}

// matches lists games for result entry: ?view=awaiting|recent&league=epl.
func (h *Handler) matches(c fiber.Ctx) error {
	m, err := h.admin.Matches(c.Context(), c.Query("view"), c.Query("league"))
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(m)
}

type resultRequest struct {
	HomeGoals *int `json:"home_goals"`
	AwayGoals *int `json:"away_goals"`
}

func (h *Handler) enterResult(c fiber.Ctx) error {
	var req resultRequest
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	if req.HomeGoals == nil || req.AwayGoals == nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "Give both home_goals and away_goals.")
	}
	if err := h.admin.EnterResult(c.Context(), current(c), c.Params("id"), *req.HomeGoals, *req.AwayGoals); err != nil {
		return apierr.From(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) thresholds(c fiber.Ctx) error { return c.JSON(h.admin.Thresholds()) }
