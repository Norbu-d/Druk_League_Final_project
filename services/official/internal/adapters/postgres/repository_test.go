package postgres

import (
	"context"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/official/internal/domain"
)

const (
	match1     = "11111111-1111-4111-8111-111111111111"
	match2     = "22222222-2222-4222-8222-222222222222"
	officialA  = "33333333-3333-4333-8333-333333333333"
	officialB  = "44444444-4444-4444-8444-444444444444"
	tournament = "55555555-5555-4555-8555-555555555555"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	first, err := repo.Record(ctx, domain.Assignment{MatchID: match1, OfficialUserID: officialA, TournamentID: tournament, AssignedBy: officialB})
	if err != nil || first.TournamentID != tournament || first.AssignedBy != officialB {
		t.Fatalf("record: %+v %v", first, err)
	}
	// Re-assigning a match replaces the official (one assignment per match).
	if _, err := repo.Record(ctx, domain.Assignment{MatchID: match1, OfficialUserID: officialB, TournamentID: tournament}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Record(ctx, domain.Assignment{MatchID: match2, OfficialUserID: officialB}); err != nil {
		t.Fatal(err)
	}
	all, err := repo.List(ctx, "", "")
	if err != nil || len(all) != 2 {
		t.Fatalf("all: %v %d", err, len(all))
	}
	byOfficial, _ := repo.List(ctx, officialB, "")
	byTournament, _ := repo.List(ctx, "", tournament)
	none, _ := repo.List(ctx, officialA, "")
	if len(byOfficial) != 2 || len(byTournament) != 1 || len(none) != 0 {
		t.Fatalf("filters: %d %d %d", len(byOfficial), len(byTournament), len(none))
	}
}
