package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/auth/internal/domain"
)

const userID = "11111111-1111-4111-8111-111111111111"

func TestSessionsAndAttemptsAgainstPostgres(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	sessions, attempts := New(db), NewAttempts(db)
	ctx := context.Background()
	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	session, err := sessions.Create(ctx, userID, "hash-1", expires)
	if err != nil || session.UserID != userID || session.Revoked {
		t.Fatalf("create: %+v %v", session, err)
	}
	if got, err := sessions.ByRefreshHash(ctx, "hash-1"); err != nil || got.ID != session.ID {
		t.Fatalf("by hash: %+v %v", got, err)
	}
	if _, err := sessions.ByRefreshHash(ctx, "nope"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing hash: %v", err)
	}
	if ok, err := sessions.Rotate(ctx, session.ID, "hash-1", "hash-2", expires); err != nil || !ok {
		t.Fatalf("rotate: %v %v", ok, err)
	}
	if ok, _ := sessions.Rotate(ctx, session.ID, "hash-1", "hash-3", expires); ok {
		t.Fatal("stale hash rotated")
	}
	if err := sessions.Revoke(ctx, session.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := sessions.Get(ctx, session.ID); err != nil || !got.Revoked {
		t.Fatalf("revoked: %+v %v", got, err)
	}
	if ok, _ := sessions.Rotate(ctx, session.ID, "hash-2", "hash-4", expires); ok {
		t.Fatal("revoked session rotated")
	}
	if _, err := sessions.Get(ctx, "22222222-2222-4222-8222-222222222222"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing session: %v", err)
	}

	if a, err := attempts.Get(ctx, "admin"); err != nil || a.Failed != 0 {
		t.Fatalf("empty attempts: %+v %v", a, err)
	}
	locked := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)
	if err := attempts.Save(ctx, "admin", domain.Attempts{Failed: 2, WindowStart: time.Now(), LockedUntil: locked}); err != nil {
		t.Fatal(err)
	}
	if err := attempts.Save(ctx, "admin", domain.Attempts{Failed: 3, LockedUntil: locked}); err != nil {
		t.Fatal(err)
	}
	if a, err := attempts.Get(ctx, "admin"); err != nil || a.Failed != 3 || !a.LockedUntil.Equal(locked) || !a.WindowStart.IsZero() {
		t.Fatalf("attempts: %+v %v", a, err)
	}
	if err := attempts.Reset(ctx, "admin"); err != nil {
		t.Fatal(err)
	}
	if a, _ := attempts.Get(ctx, "admin"); a.Failed != 0 {
		t.Fatal("reset failed")
	}
}
