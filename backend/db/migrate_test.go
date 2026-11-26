package db_test

import (
	"context"
	"os"
	"testing"

	"ticketdoctor/db"
	"ticketdoctor/db/dbtest"
)

// Every migration must apply and roll back cleanly: down to nothing, then up again.
func TestMigrationsRoundTrip(t *testing.T) {
	_ = dbtest.New(t) // skips without a test database; holds the shared lock
	ctx := context.Background()
	conn, err := db.OpenSQL(ctx, os.Getenv("TD_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	m, err := db.NewMigrator(conn)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := m.GetDBVersion(ctx)
	if err != nil || latest == 0 {
		t.Fatalf("version %d, %v", latest, err)
	}
	if _, err := m.DownTo(ctx, 0); err != nil {
		t.Fatalf("roll back every migration: %v", err)
	}
	if v, _ := m.GetDBVersion(ctx); v != 0 {
		t.Fatalf("after rolling back, version = %d", v)
	}
	if _, err := db.Migrate(ctx, conn); err != nil {
		t.Fatalf("migrate up again: %v", err)
	}
	if v, _ := m.GetDBVersion(ctx); v != latest {
		t.Fatalf("after migrating up, version = %d, want %d", v, latest)
	}
	var leagues int
	if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM leagues`).Scan(&leagues); err != nil || leagues != 3 {
		t.Fatalf("leagues after re-migrating = %d, %v; want 3", leagues, err)
	}
}
