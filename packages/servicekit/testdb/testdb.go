// Package testdb gives integration tests a fresh PostgreSQL database with a service's migrations.
// Tests are skipped unless TEST_DATABASE_URL points at a server where the user may CREATE DATABASE,
// for example postgres://postgres:test@127.0.0.1:55432/postgres?sslmode=disable.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// New creates a uniquely named database, applies the migrations in dir and drops it after the test.
func New(t *testing.T, migrationsDir string) *sql.DB {
	t.Helper()
	admin := os.Getenv("TEST_DATABASE_URL")
	if admin == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	id, err := core.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	name := "it_" + strings.ReplaceAll(id, "-", "")[:16]
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server, err := sql.Open("pgx", admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = server.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		server.Close()
		t.Fatalf("create test database: %v", err)
	}
	parsed, err := url.Parse(admin)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	db, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = server.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name))
		_ = server.Close()
	})
	if err = servicekit.ApplyMigrations(ctx, db, migrationsDir); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}
