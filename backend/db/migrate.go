package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pressly/goose/v3"
)

// Migrate applies every embedded migration that hasn't run yet and returns the
// files applied. With no migration files it does nothing.
func Migrate(ctx context.Context, conn *sql.DB) ([]*goose.MigrationResult, error) {
	p, err := NewMigrator(conn)
	if errors.Is(err, goose.ErrNoMigrations) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p.Up(ctx)
}

// NewMigrator returns a goose provider over the embedded migrations.
func NewMigrator(conn *sql.DB) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, conn, Migrations())
}
