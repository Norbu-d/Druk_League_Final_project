package grpcadapter

import (
	"context"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	managerA     = "11111111-1111-4111-8111-111111111111"
	managerB     = "22222222-2222-4222-8222-222222222222"
	tournamentID = "33333333-3333-4333-8333-333333333333"
)

type teamTestState struct {
	server      *Server
	managerCtx  context.Context
	managerBCtx context.Context
	adminCtx    context.Context
	teamA       *v1.Team
	teamB       *v1.Team
}

func newTeamTestState(t *testing.T) teamTestState {
	t.Helper()
	server := New(testdb.New(t, "../../../migrations"))
	managerCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: managerA, Role: core.RoleTeamManager})
	managerBCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: managerB, Role: core.RoleTeamManager})
	adminCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "44444444-4444-4444-8444-444444444444", Role: core.RoleSuperAdmin})
	teamA := createTeamForTest(t, server, managerCtx, "Thimphu FC", "thf", "Thimphu", tournamentID)
	teamB := createTeamForTest(t, server, managerBCtx, "Paro FC", "par", "Paro", tournamentID)
	createTeamForTest(t, server, adminCtx, "Free Agents", "free", "Punakha", "")
	return teamTestState{server: server, managerCtx: managerCtx, managerBCtx: managerBCtx, adminCtx: adminCtx, teamA: teamA, teamB: teamB}
}

func createTeamForTest(t *testing.T, server *Server, ctx context.Context, name, code, district, tournament string) *v1.Team {
	t.Helper()
	response, err := server.CreateTeam(ctx, &v1.CreateTeamRequest{Name: name, ShortCode: code, Dzongkhag: district, TournamentId: tournament})
	if err != nil {
		t.Fatalf("create team %q: %v", name, err)
	}
	return response.GetTeam()
}

func TestCreateTeamAuthorizationAndValidation(t *testing.T) {
	state := newTeamTestState(t)
	if state.teamA.GetManagerId() != managerA || state.teamB.GetManagerId() != managerB {
		t.Fatalf("manager-created teams have incorrect owners: A=%q B=%q", state.teamA.GetManagerId(), state.teamB.GetManagerId())
	}
	if _, err := state.server.CreateTeam(context.Background(), &v1.CreateTeamRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated create code = %v, want unauthenticated", status.Code(err))
	}
	viewer := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: managerA, Role: core.RoleViewer})
	if _, err := state.server.CreateTeam(viewer, &v1.CreateTeamRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("viewer create code = %v, want permission denied", status.Code(err))
	}
	if _, err := state.server.CreateTeam(state.managerCtx, &v1.CreateTeamRequest{Name: " ", ShortCode: "x", Dzongkhag: "Thimphu"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid team create code = %v, want invalid argument", status.Code(err))
	}
	badManager := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "not-a-uuid", Role: core.RoleTeamManager})
	if _, err := state.server.CreateTeam(badManager, &v1.CreateTeamRequest{Name: "Bad", ShortCode: "bad", Dzongkhag: "Thimphu"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid manager create code = %v, want invalid argument", status.Code(err))
	}
	if _, err := state.server.CreateTeam(state.adminCtx, &v1.CreateTeamRequest{Name: "Bad Tournament", ShortCode: "bad-t", Dzongkhag: "Thimphu", TournamentId: "not-a-uuid"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid tournament code = %v, want invalid argument", status.Code(err))
	}
}

func TestGetTeamRestrictsManagersToOwnedTeams(t *testing.T) {
	state := newTeamTestState(t)
	if _, err := state.server.GetTeam(state.managerCtx, &v1.GetTeamRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty team id code = %v, want invalid argument", status.Code(err))
	}
	read, err := state.server.GetTeam(state.managerCtx, &v1.GetTeamRequest{Id: state.teamA.GetId()})
	if err != nil || read.GetTeam().GetManagerId() != managerA {
		t.Fatalf("get owned team: response=%+v err=%v", read, err)
	}
	if _, err := state.server.GetTeam(state.managerCtx, &v1.GetTeamRequest{Id: state.teamB.GetId()}); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign manager team code = %v, want not found", status.Code(err))
	}
	if _, err := state.server.GetTeam(state.managerCtx, &v1.GetTeamRequest{Id: "66666666-6666-4666-8666-666666666666"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing team code = %v, want not found", status.Code(err))
	}
	if _, err := state.server.GetTeam(state.managerCtx, &v1.GetTeamRequest{Id: "not-a-uuid"}); status.Code(err) != codes.Internal {
		t.Fatalf("invalid database id code = %v, want internal", status.Code(err))
	}
	if _, err := state.server.GetTeam(context.Background(), &v1.GetTeamRequest{Id: state.teamA.GetId()}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated get code = %v, want unauthenticated", status.Code(err))
	}
}

func TestListTeamsRestrictsManagersAndPaginates(t *testing.T) {
	state := newTeamTestState(t)
	owned, err := state.server.ListTeams(state.managerCtx, &v1.ListTeamsRequest{ManagerId: managerB, PageSize: 10})
	if err != nil || len(owned.GetTeams()) != 1 || owned.GetTeams()[0].GetManagerId() != managerA {
		t.Fatalf("manager list was not restricted to caller: response=%+v err=%v", owned, err)
	}
	firstPage, err := state.server.ListTeams(state.adminCtx, &v1.ListTeamsRequest{PageSize: 2})
	if err != nil || len(firstPage.GetTeams()) != 2 || firstPage.GetNextPageToken() != "2" {
		t.Fatalf("first team page: response=%+v err=%v", firstPage, err)
	}
	secondPage, err := state.server.ListTeams(state.adminCtx, &v1.ListTeamsRequest{PageSize: 2, PageToken: firstPage.GetNextPageToken()})
	if err != nil || len(secondPage.GetTeams()) != 1 || secondPage.GetNextPageToken() != "" {
		t.Fatalf("second team page: response=%+v err=%v", secondPage, err)
	}
	if _, err := state.server.ListTeams(context.Background(), &v1.ListTeamsRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated list code = %v, want unauthenticated", status.Code(err))
	}
	if _, err := state.server.ListTeams(state.adminCtx, &v1.ListTeamsRequest{PageToken: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid page token code = %v, want invalid argument", status.Code(err))
	}
}
