//go:build integration

// Integration tests for the PostgreSQL repository implementation. These tests
// are gated behind the "integration" build tag and require a live PostgreSQL
// instance pointed to by the DATABASE_URL environment variable. They are NOT
// run by `go test ./...` (only by `go test -tags integration ./...`).
//
// Example:
//
//	DATABASE_URL=postgres://user:pass@localhost:5432/maintenance?sslmode=disable \
//	go test -tags integration ./tests/...
package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// skipIfNoDatabase skips the test if DATABASE_URL is not set.
func skipIfNoDatabase(t *testing.T) string {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	return url
}

// TestPostgres_Smoke verifies a basic PostgreSQL round-trip: connect, run
// `SELECT 1`, and verify the result. Skipped when DATABASE_URL is not set so
// the offline `go test ./...` run remains self-contained.
func TestPostgres_Smoke(t *testing.T) {
	url := skipIfNoDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(context.Background())
	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("query: %v", err)
	}
	if one != 1 {
		t.Errorf("expected 1, got %d", one)
	}
}

// TestPostgres_MigrationsApply verifies that the migration files in
// migrations/ can be applied against a live, empty database and that the
// schema is queryable afterwards. Skipped when DATABASE_URL is not set.
func TestPostgres_MigrationsApply(t *testing.T) {
	url := skipIfNoDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(context.Background())
	// Read and execute the up migration SQL. Re-applying must be safe because
	// every statement is IF NOT EXISTS / idempotent.
	upSQL, err := os.ReadFile("../migrations/0001_init.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := conn.Exec(ctx, string(upSQL)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	// Verify the places table is queryable.
	var tableName string
	if err := conn.QueryRow(ctx,
		`SELECT table_name FROM information_schema.tables WHERE table_name = 'places'`).Scan(&tableName); err != nil {
		t.Fatalf("query places: %v", err)
	}
	if tableName != "places" {
		t.Errorf("expected places, got %s", tableName)
	}
	// Apply the down migration to clean up.
	downSQL, err := os.ReadFile("../migrations/0001_init.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := conn.Exec(ctx, string(downSQL)); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}
}
