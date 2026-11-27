package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	q      *dbsqlc.Queries
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewAuthService signs sessions with secret, valid for ttl.
func NewAuthService(db dbsqlc.DBTX, secret []byte, ttl time.Duration, now func() time.Time) *AuthService {
	return &AuthService{q: dbsqlc.New(db), secret: secret, ttl: ttl, now: now}
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

// CreateAdmin adds an admin account (from the CLI only).
func (s *AuthService) CreateAdmin(ctx context.Context, username, password string) (*Admin, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernamePattern.MatchString(username) {
		return nil, Invalid("Usernames are 3–32 characters: lowercase letters, digits, dots, dashes or underscores.")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	u, err := s.q.CreateUser(ctx, dbsqlc.CreateUserParams{Username: username, PasswordHash: hash})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nil, Invalid(fmt.Sprintf("There is already an account called %q.", username))
	}
	if err != nil {
		return nil, err
	}
	a := admin(u)
	return &a, nil
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
