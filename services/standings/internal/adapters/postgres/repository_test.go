package postgres

import (
	"context"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/standings/internal/domain"
)

const (
	tournament = "33333333-3333-4333-8333-333333333333"
	home       = "44444444-4444-4444-8444-444444444444"
	away       = "55555555-5555-4555-8555-555555555555"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	ctx := context.Background()
	homeTeam := domain.Team{ID: home, Name: "Thimphu FC", Dzongkhag: "Thimphu"}
	awayTeam := domain.Team{ID: away, Name: "Paro FC", Dzongkhag: "Paro"}

	if known, err := repo.TeamKnown(ctx, tournament, home); err != nil || known {
		t.Fatalf("known before any result: %v %v", known, err)
	}
	first := domain.MatchFinalized{EventID: "11111111-1111-4111-8111-111111111111", TournamentID: tournament, HomeTeamID: home, AwayTeamID: away, HomeScore: 2, AwayScore: 1}
	if applied, err := repo.ApplyFinalized(ctx, first, homeTeam, awayTeam); err != nil || !applied {
		t.Fatalf("apply: %v %v", applied, err)
	}
	// A duplicate delivery of the same event id must not count twice.
	if applied, err := repo.ApplyFinalized(ctx, first, homeTeam, awayTeam); err != nil || applied {
		t.Fatalf("duplicate: %v %v", applied, err)
	}
	// Known teams are passed with only their id; stored names must be kept.
	second := domain.MatchFinalized{EventID: "22222222-2222-4222-8222-222222222222", TournamentID: tournament, HomeTeamID: away, AwayTeamID: home, HomeScore: 1, AwayScore: 1}
	if _, err := repo.ApplyFinalized(ctx, second, domain.Team{ID: away}, domain.Team{ID: home}); err != nil {
		t.Fatal(err)
	}
	if known, err := repo.TeamKnown(ctx, tournament, home); err != nil || !known {
		t.Fatalf("known after result: %v %v", known, err)
	}
	table, err := repo.Get(ctx, tournament)
	if err != nil || len(table) != 2 {
		t.Fatalf("get: %v %d", err, len(table))
	}
	leader := table[0]
	if leader.TeamName != "Thimphu FC" || leader.Played != 2 || leader.Won != 1 || leader.Drawn != 1 || leader.Points != 4 || leader.GoalDifference != 1 {
		t.Fatalf("unexpected leader %+v", leader)
	}
	if table[1].TeamName != "Paro FC" || table[1].Points != 1 || table[1].Lost != 1 {
		t.Fatalf("unexpected second %+v", table[1])
	}
}
