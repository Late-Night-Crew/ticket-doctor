package services

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"ticketdoctor/database/dbsqlc"
)

// Admin sign-in (B6, open question 6), in the style of nsms and parrot: bcrypt
// passwords and an HS256 JWT. The API puts the token in an HttpOnly cookie, so
// scripts on the page can never read it.

const (
	// MinPasswordLength is long enough to make guessing hopeless; bcrypt only
	// reads the first 72 bytes, so longer passwords are refused rather than
	// silently cut short.
	MinPasswordLength = 12
	maxPasswordBytes  = 72
	bcryptCost        = 12

	issuer = "ticketdoctor"
	// passwordSetClaim carries when the password was last set (as in parrot),
	// so changing a password ends every older session.
	passwordSetClaim = "pwd"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)

// ErrSignIn is the one message for every failed sign-in, so it never reveals
// whether the username exists.
var ErrSignIn = Unauthorized("Wrong username or password.")

var errSession = Unauthorized("Please sign in again.")

// dummyHash is compared against when the username doesn't exist, so a wrong
// username takes as long as a wrong password.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not a real password, only for timing"), bcryptCost)

type Admin struct {
	ID        int64      `json:"id,string"`
	Username  string     `json:"username"`
	Role      string     `json:"role"`
	LastLogin *time.Time `json:"last_login_at"`
}

type Session struct {
	Admin     Admin
	Token     string
	ExpiresAt time.Time
}

type AuthService struct {
	pool   *pgxpool.Pool
	q      *dbsqlc.Queries
	secret []byte
	ttl    time.Duration
	now    func() time.Time
	// setupCode unlocks first-admin setup; empty turns setup off.
	setupCode string
}

// NewAuthService signs sessions with secret, valid for ttl.
func NewAuthService(pool *pgxpool.Pool, secret []byte, ttl time.Duration, now func() time.Time) *AuthService {
	return &AuthService{pool: pool, q: dbsqlc.New(pool), secret: secret, ttl: ttl, now: now}
}

// WithSetupCode turns on first-admin setup with this code (TD_ADMIN_SETUP_CODE).
func (s *AuthService) WithSetupCode(code string) *AuthService {
	s.setupCode = code
	return s
}

// Login checks a username and password and starts a session.
func (s *AuthService) Login(ctx context.Context, username, password string) (*Session, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	u, err := s.q.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, ErrSignIn
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil || u.Disabled {
		return nil, ErrSignIn
	}
	if err := s.q.RecordLogin(ctx, u.ID); err != nil {
		return nil, err
	}
	return s.session(u)
}

// session signs a new session for an account.
func (s *AuthService) session(u dbsqlc.User) (*Session, error) {
	now := s.now()
	expires := now.Add(s.ttl)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":            issuer,
		"sub":            strconv.FormatInt(u.ID, 10),
		"iat":            now.Unix(),
		"exp":            expires.Unix(),
		passwordSetClaim: u.PasswordChangedAt.Unix(),
	}).SignedString(s.secret)
	if err != nil {
		return nil, err
	}
	return &Session{Admin: admin(u), Token: token, ExpiresAt: expires}, nil
}

// Verify checks a session token and returns its admin. The account is read
// each time, so disabling it or changing its password takes effect at once.
func (s *AuthService) Verify(ctx context.Context, token string) (*Admin, error) {
	if token == "" {
		return nil, Unauthorized("Please sign in.")
	}
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil {
		return nil, errSession
	}
	sub, _ := claims.GetSubject()
	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return nil, errSession
	}
	pwd, ok := claims[passwordSetClaim].(float64)
	if !ok {
		return nil, errSession
	}
	u, err := s.q.GetUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errSession
	}
	if err != nil {
		return nil, err
	}
	if u.Disabled || u.PasswordChangedAt.Unix() != int64(pwd) {
		return nil, errSession
	}
	a := admin(u)
	return &a, nil
}

// CreateAdmin adds an admin account (from the CLI, or by another admin).
func (s *AuthService) CreateAdmin(ctx context.Context, username, password string) (*Admin, error) {
	u, err := createUser(ctx, s.q, username, password)
	if err != nil {
		return nil, err
	}
	a := admin(u)
	return &a, nil
}

func createUser(ctx context.Context, q *dbsqlc.Queries, username, password string) (dbsqlc.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernamePattern.MatchString(username) {
		return dbsqlc.User{}, Invalid("Usernames are 3–32 characters: lowercase letters, digits, dots, dashes or underscores.")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return dbsqlc.User{}, err
	}
	u, err := q.CreateUser(ctx, dbsqlc.CreateUserParams{Username: username, PasswordHash: hash})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return dbsqlc.User{}, Invalid(fmt.Sprintf("There is already an account called %q.", username))
	}
	return u, err
}

// ---------- first-admin setup ----------

// SetupOpen reports whether first-admin setup can be used: a setup code is
// configured and no account exists yet.
func (s *AuthService) SetupOpen(ctx context.Context) (bool, error) {
	if s.setupCode == "" {
		return false, nil
	}
	n, err := s.q.CountUsers(ctx)
	return n == 0, err
}

var errSetupClosed = &UserError{Status: 403, Msg: "Setup is closed: an admin account already exists. Sign in instead."}

// Setup creates the very first admin, given the setup code, and signs them in.
// Once any account exists it always refuses, so the code is useless after the
// first use even if it leaks.
func (s *AuthService) Setup(ctx context.Context, code, username, password string) (*Session, error) {
	if s.setupCode == "" {
		return nil, &UserError{Status: 503, Msg: "Setup isn't turned on for this server."}
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(s.setupCode)) != 1 {
		return nil, Unauthorized("That setup code isn't right.")
	}
	var u dbsqlc.User
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.LockUsers(ctx); err != nil {
			return err
		}
		n, err := q.CountUsers(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			return errSetupClosed
		}
		u, err = createUser(ctx, q, username, password)
		if err != nil {
			return err
		}
		return q.RecordLogin(ctx, u.ID)
	})
	if err != nil {
		return nil, err
	}
	return s.session(u)
}

// ---------- managing admins ----------

type AdminAccount struct {
	Admin
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *AuthService) Accounts(ctx context.Context) ([]AdminAccount, error) {
	users, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminAccount, 0, len(users))
	for _, u := range users {
		out = append(out, AdminAccount{Admin: admin(u), Disabled: u.Disabled, CreatedAt: u.CreatedAt})
	}
	return out, nil
}

// SetDisabledBy lets one admin turn another's account off or on. Nobody can
// disable themselves, so there is always someone left who can sign in.
func (s *AuthService) SetDisabledBy(ctx context.Context, by *Admin, username string, disabled bool) error {
	if strings.EqualFold(strings.TrimSpace(username), by.Username) {
		return Invalid("You can't disable your own account.")
	}
	return s.SetDisabled(ctx, username, disabled)
}

// ChangeOwnPassword checks the current password, sets the new one (ending
// every other session) and returns a fresh session for this browser.
func (s *AuthService) ChangeOwnPassword(ctx context.Context, by *Admin, current, next string) (*Session, error) {
	u, err := s.q.GetUser(ctx, by.ID)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)) != nil {
		return nil, Invalid("Your current password isn't right.")
	}
	if current == next {
		return nil, Invalid("Choose a password different from the current one.")
	}
	if err := s.SetPassword(ctx, u.Username, next); err != nil {
		return nil, err
	}
	if u, err = s.q.GetUser(ctx, by.ID); err != nil {
		return nil, err
	}
	return s.session(u)
}

// SetPassword replaces a password and ends the account's other sessions.
func (s *AuthService) SetPassword(ctx context.Context, username, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	n, err := s.q.SetUserPassword(ctx, dbsqlc.SetUserPasswordParams{Username: strings.ToLower(username), PasswordHash: hash})
	if err == nil && n == 0 {
		return NotFound(fmt.Sprintf("No account called %q.", username))
	}
	return err
}

// SetDisabled turns an account off (it can't sign in, and its sessions stop
// working) or back on.
func (s *AuthService) SetDisabled(ctx context.Context, username string, disabled bool) error {
	n, err := s.q.SetUserDisabled(ctx, dbsqlc.SetUserDisabledParams{Username: strings.ToLower(username), Disabled: disabled})
	if err == nil && n == 0 {
		return NotFound(fmt.Sprintf("No account called %q.", username))
	}
	return err
}

func (s *AuthService) List(ctx context.Context) ([]dbsqlc.User, error) { return s.q.ListUsers(ctx) }

func hashPassword(password string) (string, error) {
	if len([]rune(password)) < MinPasswordLength {
		return "", Invalid(fmt.Sprintf("Use at least %d characters for the password.", MinPasswordLength))
	}
	if len(password) > maxPasswordBytes {
		return "", Invalid(fmt.Sprintf("Use at most %d bytes for the password.", maxPasswordBytes))
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hash), err
}

func admin(u dbsqlc.User) Admin {
	return Admin{ID: u.ID, Username: u.Username, Role: u.Role, LastLogin: u.LastLoginAt}
}
