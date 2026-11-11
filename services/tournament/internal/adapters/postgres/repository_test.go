package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/tournament/internal/domain"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	starts := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)
	created, err := repo.Create(ctx, domain.Tournament{Name: "National League", Sport: "football", Season: "2026", Status: domain.StatusRegistrationOpen, Dzongkhag: "Thimphu", StartsAt: &starts})
	if err != nil || created.ID == "" || created.StartsAt == nil || !created.StartsAt.Equal(starts) {
		t.Fatalf("create: %+v %v", created, err)
	}
	if _, err := repo.Create(ctx, domain.Tournament{Name: "national league", Sport: "football", Season: "2026", Status: domain.StatusRegistrationOpen, Dzongkhag: "Paro"}); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("duplicate name in season: %v", err)
	}
	if _, err := repo.Create(ctx, domain.Tournament{Name: "National League", Sport: "football", Season: "2027", Status: domain.StatusRegistrationOpen, Dzongkhag: "Thimphu"}); err != nil {
		t.Fatalf("same name next season should be allowed: %v", err)
	}
	got, err := repo.Get(ctx, created.ID)
	if err != nil || got.Name != "National League" || got.Status != domain.StatusRegistrationOpen {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.Get(ctx, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	page, more, err := repo.List(ctx, "", 1, 0)
	if err != nil || len(page) != 1 || !more || page[0].Season != "2027" {
		t.Fatalf("page should be newest season first: %v %+v %v", err, page, more)
	}
	season, more, err := repo.List(ctx, "2026", 10, 0)
	if err != nil || len(season) != 1 || more {
		t.Fatalf("season filter: %v %d %v", err, len(season), more)
	}
}
