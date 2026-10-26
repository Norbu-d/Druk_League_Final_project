package grpcadapter

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/score/internal/app"
	"github.com/drukleague/platform/services/score/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedScoreServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) UpdateLiveScore(ctx context.Context, req *v1.UpdateLiveScoreRequest) (*v1.UpdateLiveScoreResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	score, err := s.service.UpdateLive(ctx, identity.ID, req.GetMatchId(), req.GetHomeScore(), req.GetAwayScore())
	if err != nil {
		return nil, translate(err)
	}
	return &v1.UpdateLiveScoreResponse{Score: toProto(score)}, nil
}
func (s *Server) FinalizeMatch(ctx context.Context, req *v1.FinalizeMatchRequest) (*v1.FinalizeMatchResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	score, err := s.service.Finalize(ctx, identity.ID, req.GetMatchId(), req.GetReason())
	if err != nil {
		return nil, translate(err)
	}
	return &v1.FinalizeMatchResponse{Score: toProto(score)}, nil
}
func (s *Server) GetScore(ctx context.Context, req *v1.GetScoreRequest) (*v1.GetScoreResponse, error) {
	score, err := s.service.Get(ctx, req.GetMatchId())
	if err != nil {
		return nil, translate(err)
	}
	return &v1.GetScoreResponse{Score: toProto(score)}, nil
}
func toProto(score domain.Score) *v1.Score {
	result := &v1.Score{MatchId: score.MatchID, HomeScore: score.HomeScore, AwayScore: score.AwayScore, Finalized: score.Finalized}
	if !score.UpdatedAt.IsZero() {
		result.UpdatedAt = timestamppb.New(score.UpdatedAt)
	}
	return result
}
func translate(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		return status.Error(codes.InvalidArgument, "score request is invalid")
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "match or score not found")
	case errors.Is(err, domain.ErrConflict):
		return status.Error(codes.AlreadyExists, "score is already finalised")
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, "only the assigned match official can change this score")
	case errors.Is(err, domain.ErrDependency):
		return status.Error(codes.Unavailable, "could not read match")
	case strings.Contains(err.Error(), "invalid input syntax for type uuid"):
		return status.Error(codes.InvalidArgument, "match id must be a UUID")
	default:
		return status.Error(codes.Internal, "score operation failed")
	}
}
