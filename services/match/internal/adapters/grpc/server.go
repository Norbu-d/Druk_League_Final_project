package grpcadapter

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/match/internal/app"
	"github.com/drukleague/platform/services/match/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedMatchServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) CreateMatch(ctx context.Context, req *v1.CreateMatchRequest) (*v1.CreateMatchResponse, error) {
	if req.GetKickoff() == nil || req.GetKickoff().CheckValid() != nil {
		return nil, status.Error(codes.InvalidArgument, "kickoff is required and must be valid")
	}
	identity, _ := servicekit.IdentityFromContext(ctx)
	match, err := s.service.Create(ctx, identity.Role, domain.Match{
		TournamentID: req.GetTournamentId(), HomeTeamID: req.GetHomeTeamId(),
		AwayTeamID: req.GetAwayTeamId(), Venue: req.GetVenue(), Kickoff: req.GetKickoff().AsTime(),
	})
	if err != nil {
		return nil, translate(err)
	}
	return &v1.CreateMatchResponse{Match: toProto(match)}, nil
}

func (s *Server) GetMatch(ctx context.Context, req *v1.GetMatchRequest) (*v1.GetMatchResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "match id is required")
	}
	match, err := s.service.Get(ctx, req.GetId())
	if err != nil {
		return nil, translate(err)
	}
	return &v1.GetMatchResponse{Match: toProto(match)}, nil
}

func (s *Server) ListMatches(ctx context.Context, req *v1.ListMatchesRequest) (*v1.ListMatchesResponse, error) {
	size, offset, err := page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	var after *time.Time
	if req.GetAfter() != nil {
		if req.GetAfter().CheckValid() != nil {
			return nil, status.Error(codes.InvalidArgument, "after timestamp is invalid")
		}
		t := req.GetAfter().AsTime()
		after = &t
	}
	items, more, err := s.service.List(ctx, req.GetTournamentId(), after, int(size), offset)
	if err != nil {
		return nil, translate(err)
	}
	response := &v1.ListMatchesResponse{Matches: make([]*v1.Match, 0, len(items))}
	for _, match := range items {
		response.Matches = append(response.Matches, toProto(match))
	}
	if more {
		response.NextPageToken = strconv.Itoa(offset + len(items))
	}
	return response, nil
}

func (s *Server) AssignOfficial(ctx context.Context, req *v1.AssignOfficialRequest) (*v1.AssignOfficialResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	match, err := s.service.AssignOfficial(ctx, identity.Role, req.GetMatchId(), req.GetOfficialId())
	if err != nil {
		return nil, translate(err)
	}
	return &v1.AssignOfficialResponse{Match: toProto(match)}, nil
}

func toProto(m domain.Match) *v1.Match {
	return &v1.Match{Id: m.ID, TournamentId: m.TournamentID, HomeTeamId: m.HomeTeamID, AwayTeamId: m.AwayTeamID, Venue: m.Venue, Kickoff: timestamppb.New(m.Kickoff), Status: m.Status, OfficialId: m.OfficialID}
}

func translate(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		return status.Error(codes.InvalidArgument, "match request is invalid")
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "match not found")
	case errors.Is(err, domain.ErrConflict):
		return status.Error(codes.AlreadyExists, "match scheduling conflict or match is finalised")
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, "super admin role required")
	case errors.Is(err, domain.ErrDependency):
		return status.Error(codes.Unavailable, "could not verify team or official")
	case strings.Contains(err.Error(), "invalid input syntax for type uuid"):
		return status.Error(codes.InvalidArgument, "ids must be UUIDs")
	default:
		return status.Error(codes.Internal, "match operation failed")
	}
}

func page(size int32, token string) (int32, int, error) {
	if size <= 0 {
		size = 50
	}
	if size > 100 {
		size = 100
	}
	offset := 0
	if token != "" {
		n, err := strconv.Atoi(token)
		if err != nil || n < 0 {
			return 0, 0, status.Error(codes.InvalidArgument, "invalid page token")
		}
		offset = n
	}
	return size, offset, nil
}
