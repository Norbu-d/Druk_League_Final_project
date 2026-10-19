package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/user/internal/domain"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	repo := New(db)
	ctx := context.Background()

	created, err := repo.CreateFirstSuperAdmin(ctx, "admin", "hash-1")
	if err != nil || !created {
		t.Fatalf("bootstrap: %v %v", created, err)
	}
	if created, err = repo.CreateFirstSuperAdmin(ctx, "other-admin", "hash-2"); err != nil || created {
		t.Fatalf("second bootstrap: %v %v", created, err)
	}
	official, err := repo.Create(ctx, domain.User{Username: "karma", Role: core.RoleMatchOfficial}, "hash-3")
	if err != nil || official.ID == "" {
		t.Fatalf("create: %+v %v", official, err)
	}
	if _, err := repo.Create(ctx, domain.User{Username: "KARMA", Role: core.RoleViewer}, "x"); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("case-insensitive duplicate: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users(username,password_hash,role) VALUES('bad','x','dzongkhag_admin')`); err == nil {
		t.Fatal("role constraint missing")
	}
	if _, err := repo.Create(ctx, domain.User{Username: "manager", Role: core.RoleTeamManager}, "x"); err != nil {
		t.Fatalf("team manager role rejected: %v", err)
	}
	got, err := repo.Get(ctx, official.ID)
	if err != nil || got.Username != "karma" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.Get(ctx, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	user, hash, err := repo.Credentials(ctx, "Karma")
	if err != nil || user.ID != official.ID || hash != "hash-3" {
		t.Fatalf("credentials: %+v %q %v", user, hash, err)
	}
	if _, _, err := repo.Credentials(ctx, "ghost"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing credentials: %v", err)
	}
	if err := repo.UpdatePasswordHash(ctx, official.ID, "hash-4"); err != nil {
		t.Fatal(err)
	}
	if _, hash, _ = repo.Credentials(ctx, "karma"); hash != "hash-4" {
		t.Fatal("hash not updated")
	}
	page, more, err := repo.List(ctx, "", 2, 0)
	if err != nil || len(page) != 2 || !more {
		t.Fatalf("page: %v %d %v", err, len(page), more)
	}
	officials, _, err := repo.List(ctx, core.RoleMatchOfficial, 10, 0)
	if err != nil || len(officials) != 1 {
		t.Fatalf("filter: %v %d", err, len(officials))
	}
}
