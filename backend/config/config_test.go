package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func env(vars map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := vars[k]; return v, ok }
}

func writeFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadWithoutFileUsesDefaults(t *testing.T) {
	t.Chdir(t.TempDir()) // no config.yaml here
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Environment != Development || cfg.ListenAddress() != "127.0.0.1:8080" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadExplicitMissingFileFails(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected an error for a missing explicit config file")
	}
}

func TestFileOverridesDefaultsAndKeepsTheRest(t *testing.T) {
	cfg, err := Load(writeFile(t, "server:\n  port: 9090\nlogging:\n  level: debug\n"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 9090 || cfg.Logging.Level != "debug" {
		t.Fatalf("file values not applied: %+v", cfg)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Logging.Format != "text" {
		t.Fatalf("defaults lost for keys absent from the file: %+v", cfg)
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	cfg := Defaults()
	err := cfg.ApplyEnvironment(env(map[string]string{
		"TD_ENVIRONMENT":     "production",
		"TD_DATABASE_URL":    "postgres://u:p@db/td",
		"PORT":               "10000",
		"TD_ALLOWED_ORIGINS": "https://ticketdoctor.app, https://preview.ticketdoctor.app",
		"TD_TRUSTED_PROXIES": "10.0.0.0/8",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Environment != Production || cfg.Database.URL == "" || cfg.Server.Port != 10000 {
		t.Fatalf("env not applied: %+v", cfg)
	}
	if len(cfg.Server.AllowedOrigins) != 2 || cfg.Server.AllowedOrigins[1] != "https://preview.ticketdoctor.app" {
		t.Fatalf("origins not split: %q", cfg.Server.AllowedOrigins)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}

func TestDatabaseFieldsBuildConnString(t *testing.T) {
	cfg, err := Load(writeFile(t, `
database:
  host: localhost
  port: 5432
  username: postgres
  password: "p@ss:w/rd"
  name: ticketdoctor
`))
	if err != nil {
		t.Fatal(err)
	}
	want := "postgres://postgres:p%40ss%3Aw%2Frd@localhost:5432/ticketdoctor?sslmode=disable"
	if got := cfg.Database.ConnString(); got != want {
		t.Fatalf("ConnString() = %q, want %q", got, want)
	}
	if got := cfg.Database.Describe(); got != "localhost:5432/ticketdoctor" || strings.Contains(got, "p@ss") {
		t.Fatalf("Describe() = %q", got)
	}
}

func TestDatabaseURLWinsAndPasswordFromEnv(t *testing.T) {
	cfg := Defaults()
	cfg.Database.Host, cfg.Database.Name, cfg.Database.Username = "localhost", "ticketdoctor", "postgres"
	if err := cfg.ApplyEnvironment(env(map[string]string{"TD_DATABASE_PASSWORD": "secret"})); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg.Database.ConnString(), "postgres:secret@") {
		t.Fatalf("password from env not used: %q", cfg.Database.ConnString())
	}
	cfg.Database.URL = "postgres://u:p@pooler.supabase.com:5432/postgres"
	if cfg.Database.ConnString() != cfg.Database.URL {
		t.Fatal("URL should win over the fields")
	}
}

func TestNoDatabaseConfigured(t *testing.T) {
	if got := Defaults().Database.ConnString(); got != "" {
		t.Fatalf("defaults should have no database, got %q", got)
	}
}

func TestTDPortWinsOverRenderPort(t *testing.T) {
	cfg := Defaults()
	if err := cfg.ApplyEnvironment(env(map[string]string{"PORT": "10000", "TD_PORT": "7000"})); err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 7000 {
		t.Fatalf("port = %d, want 7000", cfg.Server.Port)
	}
}

func TestBadPortInEnvironment(t *testing.T) {
	if err := Defaults().ApplyEnvironment(env(map[string]string{"PORT": "eighty"})); err == nil {
		t.Fatal("expected an error for a non-numeric port")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*Config)
		wantErr string
	}{
		{"defaults are valid", func(*Config) {}, ""},
		{"unknown environment", func(c *Config) { c.Environment = "STAGING" }, "environment must be"},
		{"bad log level", func(c *Config) { c.Logging.Level = "loud" }, "logging level"},
		{"bad trusted proxy", func(c *Config) { c.Server.TrustedProxies = []string{"not-an-ip"} }, "trusted proxy"},
		{"production without database", func(c *Config) { c.Environment = Production }, "TD_DATABASE_URL"},
		{"production with wildcard origin", func(c *Config) {
			c.Environment = Production
			c.Database.URL = "postgres://x"
			c.Server.AllowedOrigins = []string{"*"}
		}, `not "*"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.change(cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}
