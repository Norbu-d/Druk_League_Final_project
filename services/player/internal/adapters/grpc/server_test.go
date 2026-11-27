package grpcadapter

import (
	"context"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/player/internal/app"
	"github.com/drukleague/platform/services/player/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	playerID  = "11111111-1111-4111-8111-111111111111"
	teamID    = "22222222-2222-4222-8222-222222222222"
	managerID = "66666666-6666-4666-8666-666666666666"
)

type repo struct{}

func (repo) Create(_ context.Context, p domain.Player) (domain.Player, error) {
	p.ID = playerID
	return p, nil
}
func (repo) Get(_ context.Context, id string) (domain.Player, error) {
	return domain.Player{ID: id, TeamID: teamID, GivenName: "Sonam"}, nil
}
func (repo) List(context.Context, string, int, int) ([]domain.Player, bool, error) {
	return []domain.Player{{ID: playerID}, {ID: playerID}}, true, nil
}
func (repo) Update(_ context.Context, p domain.Player) (domain.Player, error) { return p, nil }

type teams struct{}

func (teams) Get(_ context.Context, id string) (domain.TeamRef, error) {
	return domain.TeamRef{ID: id, ManagerID: managerID}, nil
}

func as(id, role string) context.Context {
	return servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: id, Role: role})
}

func TestPlayerRPCs(t *testing.T) {
	s := New(app.New(repo{}, teams{}))
	manager := as(managerID, core.RoleTeamManager)
	created, err := s.CreatePlayer(manager, &v1.CreatePlayerRequest{TeamId: teamID, GivenName: "Karma", ShirtNumber: "07"})
	if err != nil || created.GetPlayer().GetShirtNumber() != "7" || created.GetPlayer().GetId() != playerID {
		t.Fatalf("create: %v %+v", err, created)
	}
	if _, err := s.CreatePlayer(context.Background(), &v1.CreatePlayerRequest{TeamId: teamID, GivenName: "Karma"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("anonymous create: %v", err)
	}
	if _, err := s.CreatePlayer(as("o", core.RoleMatchOfficial), &v1.CreatePlayerRequest{TeamId: teamID, GivenName: "Karma"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("official create: %v", err)
	}
	updated, err := s.UpdatePlayer(manager, &v1.UpdatePlayerRequest{Id: playerID, GivenName: "Sonam", FamilyName: "Choki", ShirtNumber: "10"})
	if err != nil || updated.GetPlayer().GetFamilyName() != "Choki" || updated.GetPlayer().GetTeamId() != teamID {
		t.Fatalf("update: %v %+v", err, updated)
	}
	if _, err := s.UpdatePlayer(context.Background(), &v1.UpdatePlayerRequest{Id: playerID, GivenName: "Sonam"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("anonymous update: %v", err)
	}
	if _, err := s.UpdatePlayer(manager, &v1.UpdatePlayerRequest{Id: playerID, GivenName: "Sonam", ShirtNumber: "100"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid update: %v", err)
	}
	if got, err := s.GetPlayer(as("v", core.RoleViewer), &v1.GetPlayerRequest{Id: playerID}); err != nil || got.GetPlayer().GetGivenName() != "Sonam" {
		t.Fatalf("get: %v", err)
	}
	if _, err := s.GetPlayer(as("v", core.RoleViewer), &v1.GetPlayerRequest{Id: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad id: %v", err)
	}
	list, err := s.ListPlayers(as("v", core.RoleViewer), &v1.ListPlayersRequest{TeamId: teamID, PageSize: 2})
	if err != nil || len(list.GetPlayers()) != 2 || list.GetNextPageToken() != "2" {
		t.Fatalf("list: %v %+v", err, list)
	}
	if _, err := s.ListPlayers(as("v", core.RoleViewer), &v1.ListPlayersRequest{PageToken: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad token accepted")
	}
	if _, err := s.ListPlayers(as("v", core.RoleViewer), &v1.ListPlayersRequest{TeamId: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("bad team filter accepted")
	}
}
