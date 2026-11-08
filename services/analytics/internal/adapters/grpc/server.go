package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/analytics/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/analytics/internal/app"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedAnalyticsServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) GetDashboardMetrics(ctx context.Context, request *v1.GetDashboardMetricsRequest) (*v1.GetDashboardMetricsResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	metrics, err := s.service.DashboardMetrics(ctx, request.GetSeason(), request.GetDzongkhag())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.GetDashboardMetricsResponse{Tournaments: metrics.Tournaments, Teams: metrics.Teams, Matches: metrics.Matches, CompletedMatches: metrics.Matches}, nil
}
