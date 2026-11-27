// Package config loads config.yaml with viper, applies TD_* environment
// overrides for secrets, and validates the result. Later milestones add the
// rule thresholds here (NFR-6: one place, admin-tunable).
//
// Precedence, lowest to highest: built-in defaults, config.yaml, environment.
// config.yaml is optional, so a container can run on environment variables alone.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"ticketdoctor/engine"
)

const (
	Development = "DEVELOPMENT"
	Production  = "PRODUCTION"
)

type Config struct {
	// Environment is DEVELOPMENT (default) or PRODUCTION. PRODUCTION refuses to
	// start without a database URL and an explicit list of allowed origins.
	Environment string         `mapstructure:"environment"`
	Server      ServerConfig   `mapstructure:"server"`
	Database    DatabaseConfig `mapstructure:"database"`
	Logging     LoggingConfig  `mapstructure:"logging"`
	// Rules holds the rule thresholds (NFR-6). Defaults are the SRS values.
	Rules RulesConfig `mapstructure:"rules"`
	// FootballDataOrg is the upcoming-fixtures source (decision D4).
	FootballDataOrg FootballDataOrgConfig `mapstructure:"football_data_org"`
	// Admin configures admin sign-in (B6).
	Admin AdminConfig `mapstructure:"admin"`
}

// AdminConfig configures admin sessions. Accounts themselves are made with
// `ticketdoctor admin create`, never in config.
type AdminConfig struct {
	// SessionSecret signs admin sessions; anyone who has it can forge one. Set
	// it with TD_ADMIN_SESSION_SECRET (at least 32 characters, e.g. from
	// `openssl rand -base64 48`). Changing it signs every admin out. Without
	// it, development makes a random one at start-up (sessions end on restart)
	// and production turns admin sign-in off.
	SessionSecret string `mapstructure:"session_secret"`
	// SessionHours is how long a sign-in lasts.
	SessionHours int `mapstructure:"session_hours"`
	// SetupCode unlocks /admin/setup, which creates the first admin account
	// from the browser. It only works while there are no accounts at all, so
	// it can't be used once someone has signed up. Set it with
	// TD_ADMIN_SETUP_CODE (at least 16 characters); leave it empty to make
	// the first account with `ticketdoctor admin create` instead.
	SetupCode string `mapstructure:"setup_code"`
}

// MinSetupCode is the shortest setup code accepted.
const MinSetupCode = 16

// MinSessionSecret is the shortest session secret accepted.
const MinSessionSecret = 32

// FootballDataOrgConfig configures the fixtures import from football-data.org.
type FootballDataOrgConfig struct {
	// Token is the API key. Keep it out of committed files: put it in the
	// untracked config.yaml locally, and TD_FOOTBALL_DATA_ORG_TOKEN elsewhere.
	Token string `mapstructure:"token"`
	// DaysAhead is how far ahead to import fixtures. 14 covers today, this
	// weekend and next weekend.
	DaysAhead int `mapstructure:"days_ahead"`
}

// RulesConfig mirrors engine.Thresholds with config file names. Change these
// with care: they change every estimate and suggestion the app makes.
type RulesConfig struct {
	FormWindow         int     `mapstructure:"form_window"`
	LowConfidenceBelow int     `mapstructure:"low_confidence_below"`
	LowFrom            float64 `mapstructure:"low_from"`
	MediumFrom         float64 `mapstructure:"medium_from"`
	SwapMinGain        float64 `mapstructure:"swap_min_gain"`
	MaxSuggestions     int     `mapstructure:"max_suggestions"`
	ValueMargin        float64 `mapstructure:"value_margin"`
	BoldFloor          float64 `mapstructure:"bold_floor"`
	MaxEstimate        float64 `mapstructure:"max_estimate"`
}

// Thresholds converts the config section for the engine.
func (r RulesConfig) Thresholds() engine.Thresholds {
	return engine.Thresholds{
		FormWindow:         r.FormWindow,
		LowConfidenceBelow: r.LowConfidenceBelow,
		LowFrom:            r.LowFrom,
		MediumFrom:         r.MediumFrom,
		SwapMinGain:        r.SwapMinGain,
		MaxSuggestions:     r.MaxSuggestions,
		ValueMargin:        r.ValueMargin,
		BoldFloor:          r.BoldFloor,
		MaxEstimate:        r.MaxEstimate,
	}
}

func rulesFrom(t engine.Thresholds) RulesConfig {
	return RulesConfig{
		FormWindow:         t.FormWindow,
		LowConfidenceBelow: t.LowConfidenceBelow,
		LowFrom:            t.LowFrom,
		MediumFrom:         t.MediumFrom,
		SwapMinGain:        t.SwapMinGain,
		MaxSuggestions:     t.MaxSuggestions,
		ValueMargin:        t.ValueMargin,
		BoldFloor:          t.BoldFloor,
		MaxEstimate:        t.MaxEstimate,
	}
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	// AllowedOrigins are the frontend origins allowed to call the API (CORS).
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	// TrustedProxies are reverse proxies (IPs or CIDR ranges) in front of the API,
	// so the client IP is read from X-Forwarded-For (needed for per-IP rate limits).
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

// DatabaseConfig connects to Postgres either with separate fields (local
// development, same style as nsms) or with one connection string (Render).
// When URL is set it wins and the fields are ignored.
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	// Password can come from TD_DATABASE_PASSWORD instead of the file.
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	// Sslmode is disable for a local database, require for a remote one.
	Sslmode string `mapstructure:"sslmode"`

	// URL is a full connection string, e.g. from Render. Set it with
	// TD_DATABASE_URL rather than in the file: it contains the password.
	URL string `mapstructure:"url"`

	// MaxConns caps the connection pool. Free database plans allow few connections.
	MaxConns int32 `mapstructure:"max_conns"`
}

// ConnString returns the connection string to use: URL if set, otherwise one
// built from the fields. It is empty when no database is configured.
func (d DatabaseConfig) ConnString() string {
	if d.URL != "" {
		return d.URL
	}
	if d.Host == "" || d.Name == "" {
		return ""
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.Username, d.Password), // escapes special characters
		Host:   net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		Path:   "/" + d.Name,
	}
	if d.Sslmode != "" {
		u.RawQuery = url.Values{"sslmode": {d.Sslmode}}.Encode()
	}
	return u.String()
}

// Describe names the database for logs without revealing the password.
func (d DatabaseConfig) Describe() string {
	conn := d.ConnString()
	if conn == "" {
		return "(not configured)"
	}
	u, err := url.Parse(conn)
	if err != nil {
		return "(unreadable connection string)"
	}
	return u.Host + u.Path
}

type LoggingConfig struct {
	// Level is debug, info, warn or error.
	Level string `mapstructure:"level"`
	// Format is text (readable, for development) or json (for log collectors).
	Format string `mapstructure:"format"`
}

// Defaults are safe for local development.
func Defaults() *Config {
	return &Config{
		Environment: Development,
		Server: ServerConfig{
			Host:           "127.0.0.1",
			Port:           8080,
			AllowedOrigins: []string{"http://localhost:5173"},
		},
		Database:        DatabaseConfig{Port: 5432, Sslmode: "disable", MaxConns: 5},
		Logging:         LoggingConfig{Level: "info", Format: "text"},
		Rules:           rulesFrom(engine.DefaultThresholds()),
		FootballDataOrg: FootballDataOrgConfig{DaysAhead: 14},
		Admin:           AdminConfig{SessionHours: 12},
	}
}

// Load reads the config. With path empty it looks for ./config.yaml and carries
// on with defaults if there is none; an explicit path must exist.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	v := viper.New()
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
	}
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if path != "" || !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	// Keys present in the file overwrite the defaults; absent keys keep them.
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.ApplyEnvironment(os.LookupEnv); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ApplyEnvironment overrides settings from environment variables. lookup is
// os.LookupEnv in production and a map in tests.
//
//	TD_ENVIRONMENT        DEVELOPMENT or PRODUCTION
//	TD_DATABASE_URL       full Postgres connection string, e.g. Render's (secret)
//	TD_DATABASE_PASSWORD  password for the host/port/name fields (secret)
//	TD_HOST, TD_PORT      listen address; PORT (set by Render) also works
//	TD_ALLOWED_ORIGINS    comma-separated frontend origins
//	TD_TRUSTED_PROXIES    comma-separated IPs or CIDR ranges
//	TD_LOG_LEVEL, TD_LOG_FORMAT
//	TD_FOOTBALL_DATA_ORG_TOKEN  fixtures API key (secret)
//	TD_ADMIN_SESSION_SECRET     signs admin sessions (secret)
//	TD_ADMIN_SETUP_CODE         unlocks first-admin setup in the browser (secret)
func (c *Config) ApplyEnvironment(lookup func(string) (string, bool)) error {
	str := func(key string, dst *string) {
		if v, ok := lookup(key); ok && v != "" {
			*dst = v
		}
	}
	list := func(key string, dst *[]string) {
		if v, ok := lookup(key); ok && v != "" {
			*dst = splitList(v)
		}
	}

	str("TD_ENVIRONMENT", &c.Environment)
	str("TD_DATABASE_URL", &c.Database.URL)
	str("TD_DATABASE_PASSWORD", &c.Database.Password)
	str("TD_HOST", &c.Server.Host)
	str("TD_LOG_LEVEL", &c.Logging.Level)
	str("TD_LOG_FORMAT", &c.Logging.Format)
	str("TD_FOOTBALL_DATA_ORG_TOKEN", &c.FootballDataOrg.Token)
	str("TD_ADMIN_SESSION_SECRET", &c.Admin.SessionSecret)
	str("TD_ADMIN_SETUP_CODE", &c.Admin.SetupCode)
	list("TD_ALLOWED_ORIGINS", &c.Server.AllowedOrigins)
	list("TD_TRUSTED_PROXIES", &c.Server.TrustedProxies)

	for _, key := range []string{"PORT", "TD_PORT"} { // TD_PORT wins over PORT
		if v, ok := lookup(key); ok && v != "" {
			port, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("%s must be a number, got %q", key, v)
			}
			c.Server.Port = port
		}
	}
	c.Environment = strings.ToUpper(strings.TrimSpace(c.Environment))
	return nil
}

// Validate rejects settings that would start a broken or unsafe server.
func (c *Config) Validate() error {
	var problems []string
	if c.Environment != Development && c.Environment != Production {
		problems = append(problems, fmt.Sprintf("environment must be %s or %s, got %q", Development, Production, c.Environment))
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		problems = append(problems, fmt.Sprintf("server port must be 1-65535, got %d", c.Server.Port))
	}
	for _, p := range c.Server.TrustedProxies {
		if net.ParseIP(p) == nil {
			if _, _, err := net.ParseCIDR(p); err != nil {
				problems = append(problems, fmt.Sprintf("trusted proxy %q is not an IP or CIDR range", p))
			}
		}
	}
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		problems = append(problems, fmt.Sprintf("logging level must be debug, info, warn or error, got %q", c.Logging.Level))
	}
	if c.Logging.Format != "text" && c.Logging.Format != "json" {
		problems = append(problems, fmt.Sprintf("logging format must be text or json, got %q", c.Logging.Format))
	}
	if c.Database.URL == "" && c.Database.Host != "" && (c.Database.Port < 1 || c.Database.Port > 65535) {
		problems = append(problems, fmt.Sprintf("database port must be 1-65535, got %d", c.Database.Port))
	}
	if c.Database.MaxConns < 1 {
		problems = append(problems, "database max_conns must be at least 1")
	}
	if d := c.FootballDataOrg.DaysAhead; d < 1 || d > 31 {
		problems = append(problems, fmt.Sprintf("football_data_org days_ahead must be 1-31, got %d", d))
	}
	if s := c.Admin.SessionSecret; s != "" && len(s) < MinSessionSecret {
		problems = append(problems, fmt.Sprintf("admin session_secret must be at least %d characters", MinSessionSecret))
	}
	if s := c.Admin.SetupCode; s != "" && len(s) < MinSetupCode {
		problems = append(problems, fmt.Sprintf("admin setup_code must be at least %d characters", MinSetupCode))
	}
	if h := c.Admin.SessionHours; h < 1 || h > 24*7 {
		problems = append(problems, fmt.Sprintf("admin session_hours must be 1-168, got %d", h))
	}
	if err := c.Rules.Thresholds().Validate(); err != nil {
		problems = append(problems, "rules: "+strings.ReplaceAll(err.Error(), "\n", "; "))
	}
	if c.IsProduction() {
		if c.Database.ConnString() == "" {
			problems = append(problems, "PRODUCTION needs a database (TD_DATABASE_URL, or database host and name)")
		}
		if len(c.Server.AllowedOrigins) == 0 {
			problems = append(problems, "PRODUCTION needs allowed origins (TD_ALLOWED_ORIGINS)")
		}
		for _, o := range c.Server.AllowedOrigins {
			if o == "*" {
				problems = append(problems, `PRODUCTION must list allowed origins explicitly, not "*"`)
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid config:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func (c *Config) IsProduction() bool { return c.Environment == Production }

func (c *Config) ListenAddress() string {
	return net.JoinHostPort(c.Server.Host, strconv.Itoa(c.Server.Port))
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
