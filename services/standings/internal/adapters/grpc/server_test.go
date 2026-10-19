package grpcadapter

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	"github.com/drukleague/platform/services/standings/internal/app"
	"github.com/drukleague/platform/services/standings/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const tournament = "33333333-3333-4333-8333-333333333333"

type repo struct{ err error }

func (repo) TeamKnown(context.Context, string, string) (bool, error) { return true, nil }
func (repo) ApplyFinalized(context.Context, domain.MatchFinalized, domain.Team, domain.Team) (bool, error) {
	return true, nil
}
func (r repo) Get(context.Context, string) ([]domain.Standing, error) {
	return []domain.Standing{{TeamID: "t", TeamName: "Thimphu FC", Points: 3, GoalDifference: 1}}, r.err
}

type teams struct{}

func (teams) GetTeam(context.Context, string) (domain.Team, error) { return domain.Team{}, nil }

func TestGetStandings(t *testing.T) {
	response, err := New(app.New(repo{}, teams{})).GetStandings(context.Background(), &v1.GetStandingsRequest{TournamentId: tournament})
	if err != nil || len(response.GetStandings()) != 1 || response.GetStandings()[0].GetPoints() != 3 {
		t.Fatalf("get: %v", err)
	}
	if _, err := New(app.New(repo{}, teams{})).GetStandings(context.Background(), &v1.GetStandingsRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("missing tournament accepted")
	}
	if _, err := New(app.New(repo{err: errors.New("db")}, teams{})).GetStandings(context.Background(), &v1.GetStandingsRequest{TournamentId: tournament}); status.Code(err) != codes.Internal {
		t.Fatal("db error not Internal")
	}
	if status.Code(translate(domain.ErrDependency)) != codes.Unavailable {
		t.Fatal("dependency error not Unavailable")
	}
}
