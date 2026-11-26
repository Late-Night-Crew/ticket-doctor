// Package dbtest gives integration tests a migrated, empty Postgres database.
//
// Tests using it are skipped unless TD_TEST_DATABASE_URL is set, so `go test`
// works without a database. CI sets it (see .github/workflows/backend.yml);
// locally point it at a separate database, never the one you develop on:
//
//	export TD_TEST_DATABASE_URL="postgres://postgres:PASSWORD@localhost:5432/ticketdoctor_test?sslmode=disable"
package dbtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ticketdoctor/db"
)

const envVar = "TD_TEST_DATABASE_URL"

// lockID serialises test packages that share the test database: go test runs
// packages in parallel, and each one empties the tables.
const lockID = 74_265_001

// New returns a pool on the test database with all migrations applied and all
// data tables emptied (the reference leagues stay). It holds a lock until the
// test ends, so tests in other packages wait instead of clashing.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv(envVar)
	if url == "" {
		// On CI (GitHub sets CI=true) a missing database is a broken setup, not a
		// reason to skip: otherwise database tests could silently stop running.
		if os.Getenv("CI") != "" {
			t.Fatalf("%s is not set on CI", envVar)
		}
		t.Skipf("%s not set; skipping database test", envVar)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := db.OpenPool(ctx, url, 4)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	lock, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = lock.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lockID)
		lock.Release()
	})

	sqlDB, err := db.OpenSQL(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := db.Migrate(ctx, sqlDB); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE team_profiles, matches, teams, import_runs RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("empty test database: %v", err)
	}
	return pool
}
