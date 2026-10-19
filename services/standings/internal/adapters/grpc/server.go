package grpcadapter

import (
	"context"
	"errors"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	"github.com/drukleague/platform/services/standings/internal/app"
	"github.com/drukleague/platform/services/standings/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	v1.UnimplementedStandingsServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }
func (s *Server) GetStandings(ctx context.Context, req *v1.GetStandingsRequest) (*v1.GetStandingsResponse, error) {
	items, err := s.service.Get(ctx, req.GetTournamentId())
	if err != nil {
		return nil, translate(err)
	}
	response := &v1.GetStandingsResponse{Standings: make([]*v1.Standing, 0, len(items))}
	for _, item := range items {
		response.Standings = append(response.Standings, &v1.Standing{TeamId: item.TeamID, TeamName: item.TeamName, Dzongkhag: item.Dzongkhag, Played: item.Played, Won: item.Won, Drawn: item.Drawn, Lost: item.Lost, GoalsFor: item.GoalsFor, GoalsAgainst: item.GoalsAgainst, GoalDifference: item.GoalDifference, Points: item.Points})
	}
	return response, nil
}
func translate(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		return status.Error(codes.InvalidArgument, "a valid tournament id is required")
	case errors.Is(err, domain.ErrDependency):
		return status.Error(codes.Unavailable, "could not resolve team details")
	default:
		return status.Error(codes.Internal, "standings operation failed")
	}
}
