package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/team/internal/domain"
)

const (
	cupA    = "22222222-2222-4222-8222-222222222222"
	cupB    = "33333333-3333-4333-8333-333333333333"
	manager = "55555555-5555-4555-8555-555555555555"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	created, err := repo.Create(ctx, domain.Team{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Paro", TournamentID: cupA, ManagerID: manager})
	if err != nil || created.ID == "" || created.CreatedAt.IsZero() || created.ManagerID != manager {
		t.Fatalf("create: %+v %v", created, err)
	}
	if _, err := repo.Create(ctx, domain.Team{Name: "Paro United", ShortCode: "pfc", Dzongkhag: "Paro", TournamentID: cupA}); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("duplicate code in tournament: %v", err)
	}
	if _, err := repo.Create(ctx, domain.Team{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Paro", TournamentID: cupB}); err != nil {
		t.Fatalf("same code in another tournament should be allowed: %v", err)
	}
	unlinked, err := repo.Create(ctx, domain.Team{Name: "Haa FC", ShortCode: "HFC", Dzongkhag: "Haa"})
	if err != nil || unlinked.TournamentID != "" || unlinked.ManagerID != "" {
		t.Fatalf("team without tournament or manager: %+v %v", unlinked, err)
	}
	got, err := repo.Get(ctx, created.ID)
	if err != nil || got.Name != "Paro FC" || got.TournamentID != cupA {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.Get(ctx, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	page, more, err := repo.List(ctx, domain.Filter{}, 2, 0)
	if err != nil || len(page) != 2 || !more {
		t.Fatalf("page: %v %d %v", err, len(page), more)
	}
	for name, c := range map[string]struct {
		filter domain.Filter
		want   int
	}{
		"tournament": {domain.Filter{TournamentID: cupA}, 1},
		"dzongkhag":  {domain.Filter{Dzongkhag: "Paro"}, 2},
		"manager":    {domain.Filter{ManagerID: manager}, 1},
	} {
		items, more, err := repo.List(ctx, c.filter, 10, 0)
		if err != nil || len(items) != c.want || more {
			t.Fatalf("%s filter: %v %d %v", name, err, len(items), more)
		}
	}
}
