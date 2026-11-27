package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/player/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/player/internal/app"
	"github.com/drukleague/platform/services/player/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedPlayerServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) CreatePlayer(ctx context.Context, req *v1.CreatePlayerRequest) (*v1.CreatePlayerResponse, error) {
	caller, err := callerFrom(ctx)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	p, err := s.service.Create(ctx, caller, domain.Player{TeamID: req.GetTeamId(), GivenName: req.GetGivenName(), FamilyName: req.GetFamilyName(), ShirtNumber: req.GetShirtNumber()})
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.CreatePlayerResponse{Player: toProto(p)}, nil
}

func (s *Server) GetPlayer(ctx context.Context, req *v1.GetPlayerRequest) (*v1.GetPlayerResponse, error) {
	p, err := s.service.Get(ctx, req.GetId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.GetPlayerResponse{Player: toProto(p)}, nil
}

func (s *Server) ListPlayers(ctx context.Context, req *v1.ListPlayersRequest) (*v1.ListPlayersResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	items, more, err := s.service.List(ctx, req.GetTeamId(), size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListPlayersResponse{Players: make([]*v1.Player, 0, len(items)), NextPageToken: servicekit.NextPageToken(offset, len(items), more)}
	for _, p := range items {
		response.Players = append(response.Players, toProto(p))
	}
	return response, nil
}

func (s *Server) UpdatePlayer(ctx context.Context, req *v1.UpdatePlayerRequest) (*v1.UpdatePlayerResponse, error) {
	caller, err := callerFrom(ctx)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	p, err := s.service.Update(ctx, caller, domain.Player{ID: req.GetId(), GivenName: req.GetGivenName(), FamilyName: req.GetFamilyName(), ShirtNumber: req.GetShirtNumber()})
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.UpdatePlayerResponse{Player: toProto(p)}, nil
}

func callerFrom(ctx context.Context) (app.Caller, error) {
	identity, ok := servicekit.IdentityFromContext(ctx)
	caller, err := servicekit.Caller(identity, ok)
	return app.Caller{ID: caller.ID, Role: caller.Role}, err
}

func toProto(p domain.Player) *v1.Player {
	player := &v1.Player{Id: p.ID, TeamId: p.TeamID, GivenName: p.GivenName, FamilyName: p.FamilyName, ShirtNumber: p.ShirtNumber}
	if !p.CreatedAt.IsZero() {
		player.CreatedAt = timestamppb.New(p.CreatedAt)
	}
	return player
}
