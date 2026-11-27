package grpcadapter

import (
	"context"

	reportv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/report/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/report/internal/app"
	"github.com/drukleague/platform/services/report/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	reportv1.UnimplementedReportServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) CreateReport(ctx context.Context, req *reportv1.CreateReportRequest) (*reportv1.CreateReportResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	item, err := s.service.Create(ctx, app.Caller{ID: identity.ID, Role: identity.Role}, domain.CreateRequest{
		ReportType: req.GetReportType(), TournamentID: req.GetTournamentId(), Format: req.GetFormat(),
	})
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &reportv1.CreateReportResponse{Report: toProto(item)}, nil
}

func (s *Server) GetReport(ctx context.Context, req *reportv1.GetReportRequest) (*reportv1.GetReportResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	item, err := s.service.Get(ctx, app.Caller{ID: identity.ID, Role: identity.Role}, req.GetId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &reportv1.GetReportResponse{Report: toProto(item)}, nil
}

func toProto(item domain.Report) *reportv1.ReportJob {
	return &reportv1.ReportJob{
		Id: item.ID, ReportType: item.ReportType, Status: item.Status, DownloadUrl: item.DownloadURL,
		CreatedAt: timestamppb.New(item.CreatedAt), TournamentId: item.TournamentID, Format: item.Format,
	}
}
