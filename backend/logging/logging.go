// Package logging builds the application's structured logger (log/slog) from
// config: readable text in development, JSON for log collectors.
package logging

import (
	"fmt"
	"io"
	"log/slog"

	"ticketdoctor/config"
)

// New returns a logger writing to w at the configured level and format.
func New(cfg config.LoggingConfig, w io.Writer) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("logging level: %w", err)
	}
	opts := &slog.HandlerOptions{Level: level}
	switch cfg.Format {
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("logging format %q: want text or json", cfg.Format)
	}
}
