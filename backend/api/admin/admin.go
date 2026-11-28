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
	auth        *services.AuthService
	admin       *services.AdminService
	calibration *services.CalibrationService
	// secure marks the cookie HTTPS-only (always, except local development).
	secure bool
}

func New(auth *services.AuthService, admin *services.AdminService, calibration *services.CalibrationService, secure bool) *Handler {
	return &Handler{auth: auth, admin: admin, calibration: calibration, secure: secure}
}

// Register adds the admin routes under r (which is /api).
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/admin", noStore, jsonOnly)
	// Failed sign-ins and setup attempts share one budget per address.
	// Handlers run in the order given: the limiter must come first.
	attempts := limiter.New(limiter.Config{
		Max:                    loginAttempts,
		Expiration:             loginWindow,
		SkipSuccessfulRequests: true,
		LimitReached: func(c fiber.Ctx) error {
			return problemdetail.TooManyRequests(c, "Too many sign-in attempts. Please wait 15 minutes and try again.")
		},
	})
	g.Post("/login", attempts, h.login)
	g.Post("/logout", h.logout)
	g.Get("/setup", h.setupStatus)
	g.Post("/setup", attempts, h.setup)

	signedIn := g.Group("", h.requireAdmin)
	signedIn.Get("/me", h.me)
	signedIn.Put("/me/password", h.changePassword)
	signedIn.Get("/users", h.users)
	signedIn.Post("/users", h.createUser)
	signedIn.Put("/users/:username/disabled", h.setDisabled)
	signedIn.Get("/dashboard", h.dashboard)
	signedIn.Get("/matches", h.matches)
	signedIn.Put("/matches/:id/result", h.enterResult)
	signedIn.Get("/thresholds", h.thresholds)
	signedIn.Get("/calibration", h.calibrationReport)
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
	return h.startSession(c, s)
}

// startSession sets the session cookie and answers with the account.
func (h *Handler) startSession(c fiber.Ctx, s *services.Session) error {
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

func (h *Handler) setupStatus(c fiber.Ctx) error {
	open, err := h.auth.SetupOpen(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"open": open})
}

type setupRequest struct {
	Code     string `json:"code"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// setup creates the first admin (only while there are none) and signs them in.
func (h *Handler) setup(c fiber.Ctx) error {
	var req setupRequest
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	s, err := h.auth.Setup(c.Context(), req.Code, req.Username, req.Password)
	if err != nil {
		return apierr.From(err)
	}
	return h.startSession(c, s)
}

func (h *Handler) users(c fiber.Ctx) error {
	list, err := h.auth.Accounts(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(list)
}

func (h *Handler) createUser(c fiber.Ctx) error {
	var req loginRequest
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	a, err := h.auth.CreateAdmin(c.Context(), req.Username, req.Password)
	if err != nil {
		return apierr.From(err)
	}
	return c.Status(fiber.StatusCreated).JSON(a)
}

func (h *Handler) setDisabled(c fiber.Ctx) error {
	var req struct {
		Disabled *bool `json:"disabled"`
	}
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	if req.Disabled == nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "Give disabled: true or false.")
	}
	if err := h.auth.SetDisabledBy(c.Context(), current(c), c.Params("username"), *req.Disabled); err != nil {
		return apierr.From(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// changePassword sets a new password for the signed-in admin. Every other
// session ends; this browser gets a fresh one.
func (h *Handler) changePassword(c fiber.Ctx) error {
	var req struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if err := apierr.DecodeJSON(c, &req); err != nil {
		return err
	}
	s, err := h.auth.ChangeOwnPassword(c.Context(), current(c), req.Current, req.New)
	if err != nil {
		return apierr.From(err)
	}
	return h.startSession(c, s)
}

// calibrationReport: ?source=backtest|tickets&league=&market=&kind=picks|suggestions&low=1
func (h *Handler) calibrationReport(c fiber.Ctx) error {
	res, err := h.calibration.Calibration(c.Context(), services.CalibrationRequest{
		Source:               services.CalibrationSource(c.Query("source")),
		League:               c.Query("league"),
		Market:               c.Query("market"),
		Kind:                 c.Query("kind"),
		IncludeLowConfidence: c.Query("low") == "1",
	})
	if err != nil {
		return apierr.From(err)
	}
	return c.JSON(res)
}
