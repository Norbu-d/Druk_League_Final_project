package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/registration/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/registration/internal/app"
	"github.com/drukleague/platform/services/registration/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedRegistrationServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func caller(ctx context.Context) app.Caller {
	identity, _ := servicekit.IdentityFromContext(ctx)
	return app.Caller{ID: identity.ID, Role: identity.Role}
}

func (s *Server) RegisterTeam(ctx context.Context, req *v1.RegisterTeamRequest) (*v1.RegisterTeamResponse, error) {
	reg, err := s.service.Register(ctx, caller(ctx), req.GetTournamentId(), req.GetTeamId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.RegisterTeamResponse{Registration: toProto(reg)}, nil
}

func (s *Server) ReviewRegistration(ctx context.Context, req *v1.ReviewRegistrationRequest) (*v1.ReviewRegistrationResponse, error) {
	reg, err := s.service.Review(ctx, caller(ctx), req.GetRegistrationId(), req.GetApproved(), req.GetReason())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.ReviewRegistrationResponse{Registration: toProto(reg)}, nil
}

func (s *Server) ListRegistrations(ctx context.Context, req *v1.ListRegistrationsRequest) (*v1.ListRegistrationsResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	items, more, err := s.service.List(ctx, caller(ctx), req.GetTournamentId(), req.GetStatus(), size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListRegistrationsResponse{Registrations: make([]*v1.Registration, 0, len(items)), NextPageToken: servicekit.NextPageToken(offset, len(items), more)}
	for _, reg := range items {
		response.Registrations = append(response.Registrations, toProto(reg))
	}
	return response, nil
}

func toProto(r domain.Registration) *v1.Registration {
	return &v1.Registration{Id: r.ID, TournamentId: r.TournamentID, TeamId: r.TeamID, Status: r.Status, SubmittedAt: timestamppb.New(r.SubmittedAt)}
}
