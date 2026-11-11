package grpcadapter

import (
	"context"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/tournament/internal/app"
	"github.com/drukleague/platform/services/tournament/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const tournamentID = "11111111-1111-4111-8111-111111111111"

type repo struct{}

func (repo) Create(_ context.Context, t domain.Tournament) (domain.Tournament, error) {
	t.ID = tournamentID
	return t, nil
}
func (repo) Get(_ context.Context, id string) (domain.Tournament, error) {
	return domain.Tournament{ID: id, Name: "National League", Status: domain.StatusRegistrationOpen}, nil
}
func (repo) List(context.Context, string, int, int) ([]domain.Tournament, bool, error) {
	return []domain.Tournament{{ID: tournamentID}, {ID: tournamentID}}, true, nil
}

type dz struct{}

func (dz) Canonical(context.Context, string) (string, error) { return "Thimphu", nil }

func as(role string) context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "u", Role: role})
}

func TestTournamentRPCs(t *testing.T) {
	s := New(app.New(repo{}, dz{}))
	starts := time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)
	created, err := s.CreateTournament(as(core.RoleSuperAdmin), &v1.CreateTournamentRequest{Name: "National League", Sport: "football", Season: "2026", Dzongkhag: "thimphu", StartsAt: timestamppb.New(starts)})
	if err != nil || created.GetTournament().GetDzongkhag() != "Thimphu" || created.GetTournament().GetStatus() != domain.StatusRegistrationOpen || !created.GetTournament().GetStartsAt().AsTime().Equal(starts) {
		t.Fatalf("create: %v %+v", err, created)
	}
	if got, err := s.CreateTournament(as(core.RoleSuperAdmin), &v1.CreateTournamentRequest{Name: "Cup", Sport: "football", Season: "2026", Dzongkhag: "Thimphu"}); err != nil || got.GetTournament().GetStartsAt() != nil {
		t.Fatalf("create without start: %v", err)
	}
	if _, err := s.CreateTournament(as(core.RoleSuperAdmin), &v1.CreateTournamentRequest{Name: "Cup", Sport: "football", Season: "2026", Dzongkhag: "Thimphu", StartsAt: &timestamppb.Timestamp{Nanos: -1}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid start: %v", err)
	}
	if _, err := s.CreateTournament(as(core.RoleTeamManager), &v1.CreateTournamentRequest{Name: "Cup", Sport: "football", Season: "2026", Dzongkhag: "Thimphu"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("manager create: %v", err)
	}
	if got, err := s.GetTournament(as(core.RoleViewer), &v1.GetTournamentRequest{Id: tournamentID}); err != nil || got.GetTournament().GetName() != "National League" {
		t.Fatalf("get: %v", err)
	}
	if _, err := s.GetTournament(as(core.RoleViewer), &v1.GetTournamentRequest{Id: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad id: %v", err)
	}
	list, err := s.ListTournaments(as(core.RoleViewer), &v1.ListTournamentsRequest{PageSize: 2})
	if err != nil || len(list.GetTournaments()) != 2 || list.GetNextPageToken() != "2" {
		t.Fatalf("list: %v %+v", err, list)
	}
	if _, err := s.ListTournaments(as(core.RoleViewer), &v1.ListTournamentsRequest{PageToken: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad token accepted")
	}
}
