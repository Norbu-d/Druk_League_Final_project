package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"github.com/drukleague/platform/services/registration/internal/ports"
)

const (
	tournamentA = "11111111-1111-4111-8111-111111111111"
	tournamentB = "22222222-2222-4222-8222-222222222222"
	team1       = "33333333-3333-4333-8333-333333333333"
	team2       = "44444444-4444-4444-8444-444444444444"
	managerA    = "55555555-5555-4555-8555-555555555555"
	adminID     = "66666666-6666-4666-8666-666666666666"
	unknownID   = "77777777-7777-4777-8777-777777777777"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()

	first, err := repo.Create(ctx, domain.Registration{TournamentID: tournamentA, TeamID: team1, RegisteredBy: managerA})
	if err != nil || first.Status != domain.StatusPending || first.RegisteredBy != managerA || first.ID == "" {
		t.Fatalf("create: %+v %v", first, err)
	}
	if _, err := repo.Create(ctx, domain.Registration{TournamentID: tournamentA, TeamID: team1}); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("duplicate registration: got %v", err)
	}
	if _, err := repo.Create(ctx, domain.Registration{TournamentID: tournamentA, TeamID: team2}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, domain.Registration{TournamentID: tournamentB, TeamID: team1, RegisteredBy: managerA}); err != nil {
		t.Fatal(err)
	}

	approved, err := repo.Review(ctx, first.ID, domain.StatusApproved, "", adminID)
	if err != nil || approved.Status != domain.StatusApproved || approved.ReviewedBy != adminID {
		t.Fatalf("approve: %+v %v", approved, err)
	}
	if again, err := repo.Review(ctx, first.ID, domain.StatusApproved, "", adminID); err != nil || again.Status != domain.StatusApproved {
		t.Fatalf("repeating the same decision should succeed: %+v %v", again, err)
	}
	if _, err := repo.Review(ctx, first.ID, domain.StatusRejected, "changed my mind", adminID); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("different decision after review: got %v", err)
	}
	if _, err := repo.Review(ctx, unknownID, domain.StatusApproved, "", adminID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("unknown registration: got %v", err)
	}

	all, more, err := repo.List(ctx, ports.Filter{}, 10, 0)
	if err != nil || len(all) != 3 || more {
		t.Fatalf("all: %d %v %v", len(all), more, err)
	}
	byTournament, _, _ := repo.List(ctx, ports.Filter{TournamentID: tournamentA}, 10, 0)
	byStatus, _, _ := repo.List(ctx, ports.Filter{Status: domain.StatusApproved}, 10, 0)
	byManager, _, _ := repo.List(ctx, ports.Filter{RegisteredBy: managerA}, 10, 0)
	none, _, _ := repo.List(ctx, ports.Filter{TournamentID: unknownID}, 10, 0)
	if len(byTournament) != 2 || len(byStatus) != 1 || len(byManager) != 2 || len(none) != 0 {
		t.Fatalf("filters: %d %d %d %d", len(byTournament), len(byStatus), len(byManager), len(none))
	}

	page1, more, err := repo.List(ctx, ports.Filter{}, 2, 0)
	if err != nil || len(page1) != 2 || !more {
		t.Fatalf("page 1: %d %v %v", len(page1), more, err)
	}
	page2, more, err := repo.List(ctx, ports.Filter{}, 2, 2)
	if err != nil || len(page2) != 1 || more {
		t.Fatalf("page 2: %d %v %v", len(page2), more, err)
	}
}
