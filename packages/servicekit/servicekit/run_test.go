package servicekit

import (
	"strings"
	"testing"
)

func TestRunValidatesConfiguration(t *testing.T) {
	if err := Run(""); err == nil || !strings.Contains(err.Error(), "service name") {
		t.Fatalf("empty service name: %v", err)
	}
	t.Setenv("DATABASE_URL", "")
	if err := Run("test-service"); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("missing DATABASE_URL: %v", err)
	}
	// An unreachable database fails fast at ping instead of serving without storage.
	t.Setenv("DATABASE_URL", "postgres://nobody:nothing@127.0.0.1:1/none?sslmode=disable&connect_timeout=1")
	if err := Run("test-service"); err == nil || !strings.Contains(err.Error(), "connect to service database") {
		t.Fatalf("unreachable database: %v", err)
	}
}

func TestApplyMigrationsWithNoFiles(t *testing.T) {
	if err := ApplyMigrations(t.Context(), nil, t.TempDir()); err != nil {
		t.Fatalf("empty migration directory should be a no-op: %v", err)
	}
	if err := ApplyMigrations(t.Context(), nil, "["); err == nil {
		t.Fatal("malformed glob should fail")
	}
}
