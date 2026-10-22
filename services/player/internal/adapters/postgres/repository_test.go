package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/player/internal/domain"
)

const (
	teamA = "22222222-2222-4222-8222-222222222222"
	teamB = "33333333-3333-4333-8333-333333333333"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	created, err := repo.Create(ctx, domain.Player{TeamID: teamA, GivenName: "Karma", ShirtNumber: "7"})
	if err != nil || created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("create: %+v %v", created, err)
	}
	if _, err := repo.Create(ctx, domain.Player{TeamID: teamA, GivenName: "Pema", ShirtNumber: "7"}); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("duplicate shirt in team: %v", err)
	}
	if _, err := repo.Create(ctx, domain.Player{TeamID: teamB, GivenName: "Pema", ShirtNumber: "7"}); err != nil {
		t.Fatalf("same shirt in another team should be allowed: %v", err)
	}
	second, err := repo.Create(ctx, domain.Player{TeamID: teamA, GivenName: "Dorji"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, domain.Player{TeamID: teamA, GivenName: "Tashi"}); err != nil {
		t.Fatalf("players without shirt numbers should not clash: %v", err)
	}
	if _, err := repo.Update(ctx, domain.Player{ID: second.ID, GivenName: "Dorji", ShirtNumber: "7"}); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("update to a taken shirt: %v", err)
	}
	updated, err := repo.Update(ctx, domain.Player{ID: second.ID, GivenName: "Dorji", FamilyName: "Wangdi", ShirtNumber: "8"})
	if err != nil || updated.FamilyName != "Wangdi" || updated.TeamID != teamA {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if _, err := repo.Update(ctx, domain.Player{ID: "11111111-1111-4111-8111-111111111111", GivenName: "Nobody"}); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("update missing: %v", err)
	}
	if got, err := repo.Get(ctx, created.ID); err != nil || got.GivenName != "Karma" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.Get(ctx, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	page, more, err := repo.List(ctx, teamA, 2, 0)
	if err != nil || len(page) != 2 || !more {
		t.Fatalf("page: %v %d %v", err, len(page), more)
	}
	teamOnly, more, err := repo.List(ctx, teamB, 10, 0)
	if err != nil || len(teamOnly) != 1 || more {
		t.Fatalf("team filter: %v %d %v", err, len(teamOnly), more)
	}
}
