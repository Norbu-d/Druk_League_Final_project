package grpcadapter

import (
	"context"
	"testing"
	"time"

	reportv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/report/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/report/internal/app"
	"github.com/drukleague/platform/services/report/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const grpcRequester = "11111111-1111-4111-8111-111111111111"
const grpcTournament = "22222222-2222-4222-8222-222222222222"
const grpcReportID = "33333333-3333-4333-8333-333333333333"

type repoFake struct{ item domain.Report }

func (r *repoFake) Create(_ context.Context, item domain.Report) (domain.Report, error) {
	item.ID, item.CreatedAt = grpcReportID, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r.item = item
	return item, nil
}
func (r *repoFake) Get(context.Context, string) (domain.Report, error) { return r.item, nil }

type standingsFake struct{}

func (standingsFake) Get(context.Context, string) ([]domain.Standing, error) {
	return []domain.Standing{{TeamID: grpcRequester, TeamName: "Thimphu", Points: 3}}, nil
}

type fixturesFake struct{}

func (fixturesFake) List(context.Context, string, int32, string) ([]domain.Fixture, string, error) {
	return []domain.Fixture{}, "", nil
}

func TestReportRPCs(t *testing.T) {
	repo := &repoFake{}
	service := app.New(repo, standingsFake{}, fixturesFake{}, "")
	server := New(service)
	ctx := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: grpcRequester, Role: core.RoleSuperAdmin})
	created, err := server.CreateReport(ctx, &reportv1.CreateReportRequest{ReportType: "standings", TournamentId: grpcTournament})
	if err != nil || created.GetReport().GetStatus() != "complete" || created.GetReport().GetDownloadUrl() != "/api/reports/files/"+grpcReportID || created.GetReport().GetCreatedAt() == nil {
		t.Fatalf("create report response=%+v err=%v", created, err)
	}
	read, err := server.GetReport(ctx, &reportv1.GetReportRequest{Id: grpcReportID})
	if err != nil || read.GetReport().GetId() != grpcReportID {
		t.Fatalf("get report response=%+v err=%v", read, err)
	}
	if _, err := server.CreateReport(context.Background(), &reportv1.CreateReportRequest{ReportType: "standings", TournamentId: grpcTournament}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("unauthenticated create code=%v", status.Code(err))
	}
}
