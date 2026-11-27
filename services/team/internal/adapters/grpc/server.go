package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/team/internal/app"
	"github.com/drukleague/platform/services/team/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedTeamServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) CreateTeam(ctx context.Context, req *v1.CreateTeamRequest) (*v1.CreateTeamResponse, error) {
	identity, ok := servicekit.IdentityFromContext(ctx)
	caller, err := servicekit.Caller(identity, ok)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	t, err := s.service.Create(ctx, app.Caller{ID: caller.ID, Role: caller.Role}, domain.Team{Name: req.GetName(), ShortCode: req.GetShortCode(), Dzongkhag: req.GetDzongkhag(), TournamentID: req.GetTournamentId()})
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.CreateTeamResponse{Team: toProto(t)}, nil
}

func (s *Server) GetTeam(ctx context.Context, req *v1.GetTeamRequest) (*v1.GetTeamResponse, error) {
	t, err := s.service.Get(ctx, req.GetId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.GetTeamResponse{Team: toProto(t)}, nil
}

func (s *Server) ListTeams(ctx context.Context, req *v1.ListTeamsRequest) (*v1.ListTeamsResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	items, more, err := s.service.List(ctx, domain.Filter{TournamentID: req.GetTournamentId(), Dzongkhag: req.GetDzongkhag(), ManagerID: req.GetManagerId()}, size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListTeamsResponse{Teams: make([]*v1.Team, 0, len(items)), NextPageToken: servicekit.NextPageToken(offset, len(items), more)}
	for _, t := range items {
		response.Teams = append(response.Teams, toProto(t))
	}
	return response, nil
}

func toProto(t domain.Team) *v1.Team {
	team := &v1.Team{Id: t.ID, Name: t.Name, ShortCode: t.ShortCode, Dzongkhag: t.Dzongkhag, TournamentId: t.TournamentID, ManagerId: t.ManagerID}
	if !t.CreatedAt.IsZero() {
		team.CreatedAt = timestamppb.New(t.CreatedAt)
	}
	return team
}
