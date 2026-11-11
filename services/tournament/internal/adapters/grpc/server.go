package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/tournament/internal/app"
	"github.com/drukleague/platform/services/tournament/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedTournamentServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) CreateTournament(ctx context.Context, req *v1.CreateTournamentRequest) (*v1.CreateTournamentResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	t := domain.Tournament{Name: req.GetName(), Sport: req.GetSport(), Season: req.GetSeason(), Venue: req.GetVenue(), Dzongkhag: req.GetDzongkhag()}
	if req.GetStartsAt() != nil {
		if err := req.GetStartsAt().CheckValid(); err != nil {
			return nil, servicekit.ToStatus(core.Invalid("starts_at is invalid"))
		}
		starts := req.GetStartsAt().AsTime()
		t.StartsAt = &starts
	}
	created, err := s.service.Create(ctx, identity.Role, t)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.CreateTournamentResponse{Tournament: toProto(created)}, nil
}

func (s *Server) GetTournament(ctx context.Context, req *v1.GetTournamentRequest) (*v1.GetTournamentResponse, error) {
	t, err := s.service.Get(ctx, req.GetId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.GetTournamentResponse{Tournament: toProto(t)}, nil
}

func (s *Server) ListTournaments(ctx context.Context, req *v1.ListTournamentsRequest) (*v1.ListTournamentsResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	items, more, err := s.service.List(ctx, req.GetSeason(), size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListTournamentsResponse{Tournaments: make([]*v1.Tournament, 0, len(items)), NextPageToken: servicekit.NextPageToken(offset, len(items), more)}
	for _, t := range items {
		response.Tournaments = append(response.Tournaments, toProto(t))
	}
	return response, nil
}

func toProto(t domain.Tournament) *v1.Tournament {
	tournament := &v1.Tournament{Id: t.ID, Name: t.Name, Sport: t.Sport, Season: t.Season, Status: t.Status, Venue: t.Venue, Dzongkhag: t.Dzongkhag}
	if t.StartsAt != nil {
		tournament.StartsAt = timestamppb.New(*t.StartsAt)
	}
	return tournament
}
