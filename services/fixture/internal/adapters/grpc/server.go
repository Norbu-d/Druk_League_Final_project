package grpcadapter

import (
	"context"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/fixture/internal/app"
	"github.com/drukleague/platform/services/fixture/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedFixtureServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) GenerateFixtures(ctx context.Context, req *v1.GenerateFixturesRequest) (*v1.GenerateFixturesResponse, error) {
	var start time.Time
	if req.GetStartsAt() != nil {
		if req.GetStartsAt().CheckValid() != nil {
			return nil, servicekit.ToStatus(core.Invalid("starts_at is invalid"))
		}
		start = req.GetStartsAt().AsTime()
	}
	identity, _ := servicekit.IdentityFromContext(ctx)
	fixtures, created, err := s.service.Generate(ctx, app.Caller{ID: identity.ID, Role: identity.Role}, req.GetTournamentId(), req.GetFormat(), start)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.GenerateFixturesResponse{Fixtures: make([]*v1.Fixture, 0, len(fixtures)), CreatedCount: int32(created)}
	for _, f := range fixtures {
		response.Fixtures = append(response.Fixtures, toProto(f))
	}
	return response, nil
}

func (s *Server) ListFixtures(ctx context.Context, req *v1.ListFixturesRequest) (*v1.ListFixturesResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	fixtures, more, err := s.service.List(ctx, req.GetTournamentId(), size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListFixturesResponse{Fixtures: make([]*v1.Fixture, 0, len(fixtures)), NextPageToken: servicekit.NextPageToken(offset, len(fixtures), more)}
	for _, f := range fixtures {
		response.Fixtures = append(response.Fixtures, toProto(f))
	}
	return response, nil
}

func toProto(f domain.Fixture) *v1.Fixture {
	return &v1.Fixture{Id: f.ID, TournamentId: f.TournamentID, HomeTeamId: f.HomeTeamID, AwayTeamId: f.AwayTeamID, VenueId: f.VenueID, Kickoff: timestamppb.New(f.Kickoff), MatchId: f.MatchID, Round: f.Round}
}
