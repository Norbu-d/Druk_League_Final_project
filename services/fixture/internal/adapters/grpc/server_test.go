package grpcadapter

import (
	"context"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/fixture/internal/app"
	"github.com/drukleague/platform/services/fixture/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const tournament = "11111111-1111-4111-8111-111111111111"

type repo struct{}

func (repo) SaveSchedule(_ context.Context, _ string, f []domain.Fixture, _ domain.OutboxEvent) ([]domain.Fixture, bool, error) {
	for i := range f {
		f[i].ID, f[i].MatchID = "f", "m"
	}
	return f, true, nil
}
func (repo) LinkMatch(context.Context, domain.Fixture) error { return nil }
func (repo) List(context.Context, string, int, int) ([]domain.Fixture, bool, error) {
	return []domain.Fixture{{ID: "f1", Round: 1}, {ID: "f2", Round: 1}}, true, nil
}

type tournaments struct{}

func (tournaments) Get(context.Context, string) (domain.Tournament, error) {
	return domain.Tournament{ID: tournament}, nil
}

type teams struct{}

func (teams) IDsInTournament(context.Context, string) ([]string, error) {
	return []string{"a", "b"}, nil
}

type venues struct{}

func (venues) InDzongkhag(context.Context, string) ([]domain.Venue, error) { return nil, nil }

type matches struct{}

func (matches) Create(context.Context, domain.Fixture) (string, error) { return "m", nil }

func server() *Server {
	return New(app.New(repo{}, tournaments{}, teams{}, venues{}, matches{}))
}

func admin() context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "a", Role: core.RoleSuperAdmin})
}

func TestGenerateFixturesRPC(t *testing.T) {
	start := time.Date(2026, 12, 1, 4, 0, 0, 0, time.UTC)
	response, err := server().GenerateFixtures(admin(), &v1.GenerateFixturesRequest{TournamentId: tournament, StartsAt: timestamppb.New(start)})
	if err != nil || response.GetCreatedCount() != 1 || len(response.GetFixtures()) != 1 {
		t.Fatalf("generate: %v %+v", err, response)
	}
	if !response.GetFixtures()[0].GetKickoff().AsTime().Equal(start) || response.GetFixtures()[0].GetMatchId() != "m" {
		t.Fatal("fixture fields not mapped")
	}
	if _, err := server().GenerateFixtures(admin(), &v1.GenerateFixturesRequest{TournamentId: tournament, StartsAt: &timestamppb.Timestamp{Nanos: -1}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid start: %v", err)
	}
	if _, err := server().GenerateFixtures(context.Background(), &v1.GenerateFixturesRequest{TournamentId: tournament}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("anonymous: %v", err)
	}
}

func TestListFixturesRPC(t *testing.T) {
	response, err := server().ListFixtures(admin(), &v1.ListFixturesRequest{TournamentId: tournament, PageSize: 2})
	if err != nil || len(response.GetFixtures()) != 2 || response.GetNextPageToken() != "2" {
		t.Fatalf("list: %v %+v", err, response)
	}
	if _, err := server().ListFixtures(admin(), &v1.ListFixturesRequest{TournamentId: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad id: %v", err)
	}
	if _, err := server().ListFixtures(admin(), &v1.ListFixturesRequest{TournamentId: tournament, PageToken: "z"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad token: %v", err)
	}
}
