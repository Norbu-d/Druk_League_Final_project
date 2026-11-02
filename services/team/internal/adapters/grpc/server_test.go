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

func TestTeamOwnerAndReadRPCs(t *testing.T) {
	db := testdb.New(t, "../../../migrations")
	server := New(db)
	managerCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: managerA, Role: core.RoleTeamManager})
	managerBContext := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: managerB, Role: core.RoleTeamManager})
	adminCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "44444444-4444-4444-8444-444444444444", Role: core.RoleSuperAdmin})

	created, err := server.CreateTeam(managerCtx, &v1.CreateTeamRequest{Name: "Thimphu FC", ShortCode: "thf", Dzongkhag: "Thimphu", TournamentId: tournamentID})
	if err != nil || created.GetTeam().GetManagerId() != managerA {
		t.Fatalf("manager-created team: response=%+v err=%v", created, err)
	}
	teamA := created.GetTeam()
	createdB, err := server.CreateTeam(managerBContext, &v1.CreateTeamRequest{Name: "Paro FC", ShortCode: "par", Dzongkhag: "Paro", TournamentId: tournamentID})
	if err != nil || createdB.GetTeam().GetManagerId() != managerB {
		t.Fatalf("second manager-created team: response=%+v err=%v", createdB, err)
	}
	adminTeam, err := server.CreateTeam(adminCtx, &v1.CreateTeamRequest{Name: "Free Agents", ShortCode: "free", Dzongkhag: "Punakha"})
	if err != nil || adminTeam.GetTeam().GetManagerId() != "" {
		t.Fatalf("admin-created team should be unassigned: response=%+v err=%v", adminTeam, err)
	}

	if _, err := server.CreateTeam(context.Background(), &v1.CreateTeamRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated create code = %v, want unauthenticated", status.Code(err))
	}
	viewerCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: managerA, Role: core.RoleViewer})
	if _, err := server.CreateTeam(viewerCtx, &v1.CreateTeamRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("viewer create code = %v, want permission denied", status.Code(err))
	}
	if _, err := server.CreateTeam(managerCtx, &v1.CreateTeamRequest{Name: " ", ShortCode: "x", Dzongkhag: "Thimphu"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid team create code = %v, want invalid argument", status.Code(err))
	}
	badManagerCtx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "not-a-uuid", Role: core.RoleTeamManager})
	if _, err := server.CreateTeam(badManagerCtx, &v1.CreateTeamRequest{Name: "Bad", ShortCode: "bad", Dzongkhag: "Thimphu"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid manager create code = %v, want invalid argument", status.Code(err))
	}
	if _, err := server.CreateTeam(adminCtx, &v1.CreateTeamRequest{Name: "Bad Tournament", ShortCode: "bad-t", Dzongkhag: "Thimphu", TournamentId: "not-a-uuid"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid tournament code = %v, want invalid argument", status.Code(err))
	}

	if _, err := server.GetTeam(managerCtx, &v1.GetTeamRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty team id code = %v, want invalid argument", status.Code(err))
	}
	read, err := server.GetTeam(managerCtx, &v1.GetTeamRequest{Id: teamA.GetId()})
	if err != nil || read.GetTeam().GetManagerId() != managerA {
		t.Fatalf("get team: response=%+v err=%v", read, err)
	}
	if _, err := server.GetTeam(managerCtx, &v1.GetTeamRequest{Id: "66666666-6666-4666-8666-666666666666"}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing team code = %v, want not found", status.Code(err))
	}
	if _, err := server.GetTeam(managerCtx, &v1.GetTeamRequest{Id: createdB.GetTeam().GetId()}); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign manager team code = %v, want not found", status.Code(err))
	}
	if _, err := server.GetTeam(managerCtx, &v1.GetTeamRequest{Id: "not-a-uuid"}); status.Code(err) != codes.Internal {
		t.Fatalf("invalid database id code = %v, want internal", status.Code(err))
	}

	owned, err := server.ListTeams(managerCtx, &v1.ListTeamsRequest{ManagerId: managerB, PageSize: 10})
	if err != nil || len(owned.GetTeams()) != 1 || owned.GetTeams()[0].GetManagerId() != managerA {
		t.Fatalf("manager list was not restricted to caller: response=%+v err=%v", owned, err)
	}
	firstPage, err := server.ListTeams(adminCtx, &v1.ListTeamsRequest{PageSize: 2})
	if err != nil || len(firstPage.GetTeams()) != 2 || firstPage.GetNextPageToken() != "2" {
		t.Fatalf("first team page: response=%+v err=%v", firstPage, err)
	}
	secondPage, err := server.ListTeams(adminCtx, &v1.ListTeamsRequest{PageSize: 2, PageToken: firstPage.GetNextPageToken()})
	if err != nil || len(secondPage.GetTeams()) != 1 || secondPage.GetNextPageToken() != "" {
		t.Fatalf("second team page: response=%+v err=%v", secondPage, err)
	}
	if _, err := server.ListTeams(context.Background(), &v1.ListTeamsRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated list code = %v, want unauthenticated", status.Code(err))
	}
	if _, err := server.ListTeams(adminCtx, &v1.ListTeamsRequest{PageToken: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid page token code = %v, want invalid argument", status.Code(err))
	}
}
