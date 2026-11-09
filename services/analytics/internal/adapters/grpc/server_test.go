package grpcadapter

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/analytics/v1"
	"github.com/drukleague/platform/services/analytics/internal/app"
	"github.com/drukleague/platform/services/analytics/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type dashboardRepository struct {
	metrics domain.DashboardMetrics
	err     error
}

func (r dashboardRepository) RecordFinalized(context.Context, domain.MatchFinalized, domain.Tournament) error {
	return nil
}
func (r dashboardRepository) DashboardMetrics(context.Context, string, string) (domain.DashboardMetrics, error) {
	return r.metrics, r.err
}

type emptyDirectory struct{}

func (emptyDirectory) GetTournament(context.Context, string) (domain.Tournament, error) {
	return domain.Tournament{}, nil
}

func TestGetDashboardMetrics(t *testing.T) {
	server := New(app.New(dashboardRepository{metrics: domain.DashboardMetrics{Tournaments: 2, Teams: 4, Matches: 7}}, emptyDirectory{}))
	if _, err := server.GetDashboardMetrics(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request err=%v", err)
	}
	response, err := server.GetDashboardMetrics(context.Background(), &v1.GetDashboardMetricsRequest{Season: "2026", Dzongkhag: "Paro"})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetTournaments() != 2 || response.GetTeams() != 4 || response.GetMatches() != 7 || response.GetCompletedMatches() != 7 {
		t.Fatalf("response=%+v", response)
	}
	server = New(app.New(dashboardRepository{err: errors.New("db down")}, emptyDirectory{}))
	if _, err := server.GetDashboardMetrics(context.Background(), &v1.GetDashboardMetricsRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("service failure err=%v", err)
	}
}
