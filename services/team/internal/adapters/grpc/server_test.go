package grpcadapter

import (
	"context"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/team/internal/app"
	"github.com/drukleague/platform/services/team/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	teamID    = "11111111-1111-4111-8111-111111111111"
	managerID = "55555555-5555-4555-8555-555555555555"
)

type repo struct{}

func (repo) Create(_ context.Context, t domain.Team) (domain.Team, error) {
	t.ID = teamID
	return t, nil
}
func (repo) Get(_ context.Context, id string) (domain.Team, error) {
	return domain.Team{ID: id, Name: "Paro FC", ManagerID: managerID}, nil
}
func (repo) List(context.Context, domain.Filter, int, int) ([]domain.Team, bool, error) {
	return []domain.Team{{ID: teamID}, {ID: teamID}}, true, nil
}

type tournaments struct{}

func (tournaments) Get(_ context.Context, id string) (domain.TournamentRef, error) {
	return domain.TournamentRef{ID: id, Status: domain.TournamentOpen}, nil
}

type dz struct{}

func (dz) Canonical(context.Context, string) (string, error) { return "Paro", nil }

func as(id, role string) context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: id, Role: role})
}

func TestTeamRPCs(t *testing.T) {
	s := New(app.New(repo{}, tournaments{}, dz{}))
	created, err := s.CreateTeam(as(managerID, core.RoleTeamManager), &v1.CreateTeamRequest{Name: "Paro FC", ShortCode: "pfc", Dzongkhag: "paro"})
	if err != nil || created.GetTeam().GetManagerId() != managerID || created.GetTeam().GetShortCode() != "PFC" {
		t.Fatalf("create: %v %+v", err, created)
	}
	if _, err := s.CreateTeam(context.Background(), &v1.CreateTeamRequest{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Paro"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("anonymous create: %v", err)
	}
	if _, err := s.CreateTeam(as("v", core.RoleViewer), &v1.CreateTeamRequest{Name: "Paro FC", ShortCode: "PFC", Dzongkhag: "Paro"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("viewer create: %v", err)
	}
	if got, err := s.GetTeam(as("a", core.RoleSuperAdmin), &v1.GetTeamRequest{Id: teamID}); err != nil || got.GetTeam().GetManagerId() != managerID {
		t.Fatalf("get: %v", err)
	}
	if _, err := s.GetTeam(as("a", core.RoleSuperAdmin), &v1.GetTeamRequest{Id: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad id: %v", err)
	}
	list, err := s.ListTeams(as("a", core.RoleService), &v1.ListTeamsRequest{PageSize: 2})
	if err != nil || len(list.GetTeams()) != 2 || list.GetNextPageToken() != "2" {
		t.Fatalf("list: %v %+v", err, list)
	}
	if _, err := s.ListTeams(as("a", core.RoleService), &v1.ListTeamsRequest{PageToken: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad token accepted")
	}
	if _, err := s.ListTeams(as("a", core.RoleService), &v1.ListTeamsRequest{ManagerId: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad manager filter accepted")
	}
}
