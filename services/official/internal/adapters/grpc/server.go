package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/official/internal/app"
	"github.com/drukleague/platform/services/official/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedOfficialServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func caller(ctx context.Context) app.Caller {
	identity, _ := servicekit.IdentityFromContext(ctx)
	return app.Caller{ID: identity.ID, Role: identity.Role}
}

func (s *Server) AssignOfficial(ctx context.Context, req *v1.AssignOfficialRequest) (*v1.AssignOfficialResponse, error) {
	a, err := s.service.Assign(ctx, caller(ctx), req.GetMatchId(), req.GetOfficialUserId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.AssignOfficialResponse{Assignment: toProto(a)}, nil
}

func (s *Server) ListAssignments(ctx context.Context, req *v1.ListAssignmentsRequest) (*v1.ListAssignmentsResponse, error) {
	items, err := s.service.List(ctx, caller(ctx), req.GetOfficialUserId(), req.GetTournamentId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListAssignmentsResponse{Assignments: make([]*v1.OfficialAssignment, 0, len(items))}
	for _, a := range items {
		response.Assignments = append(response.Assignments, toProto(a))
	}
	return response, nil
}

func toProto(a domain.Assignment) *v1.OfficialAssignment {
	return &v1.OfficialAssignment{Id: a.ID, MatchId: a.MatchID, OfficialUserId: a.OfficialUserID, AssignedAt: timestamppb.New(a.AssignedAt)}
}
