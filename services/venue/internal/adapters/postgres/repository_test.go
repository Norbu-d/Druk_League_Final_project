package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/venue/internal/domain"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	created, err := repo.Create(ctx, domain.Venue{Name: "Changlimithang Stadium", Dzongkhag: "Thimphu", Capacity: 15000})
	if err != nil || created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("create: %+v %v", created, err)
	}
	if _, err := repo.Create(ctx, domain.Venue{Name: "changlimithang stadium", Dzongkhag: "Thimphu"}); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("duplicate name in district: %v", err)
	}
	if _, err := repo.Create(ctx, domain.Venue{Name: "Changlimithang Stadium", Dzongkhag: "Paro"}); err != nil {
		t.Fatalf("same name in another district should be allowed: %v", err)
	}
	if _, err := repo.Create(ctx, domain.Venue{Name: "Punakha Ground", Dzongkhag: "Punakha"}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, created.ID)
	if err != nil || got.Capacity != 15000 {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.Get(ctx, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	page, more, err := repo.List(ctx, "", 2, 0)
	if err != nil || len(page) != 2 || !more {
		t.Fatalf("page: %v %d %v", err, len(page), more)
	}
	thimphu, more, err := repo.List(ctx, "Thimphu", 10, 0)
	if err != nil || len(thimphu) != 1 || more {
		t.Fatalf("filter: %v %d %v", err, len(thimphu), more)
	}
}
