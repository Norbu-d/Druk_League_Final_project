package grpcadapter

import (
	"context"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/user/internal/app"
	"github.com/drukleague/platform/services/user/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	v1.UnimplementedUserServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func caller(ctx context.Context) app.Caller {
	identity, _ := servicekit.IdentityFromContext(ctx)
	return app.Caller{ID: identity.ID, Role: identity.Role}
}

func (s *Server) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	user, err := s.service.Create(ctx, caller(ctx), req.GetUsername(), req.GetPassword(), req.GetRole())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.CreateUserResponse{User: toProto(user)}, nil
}

func (s *Server) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	user, err := s.service.Get(ctx, caller(ctx), req.GetId())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.GetUserResponse{User: toProto(user)}, nil
}

func (s *Server) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	size, offset, err := servicekit.Page(req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	users, more, err := s.service.List(ctx, caller(ctx), req.GetRole(), size, offset)
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	response := &v1.ListUsersResponse{Users: make([]*v1.User, 0, len(users)), NextPageToken: servicekit.NextPageToken(offset, len(users), more)}
	for _, user := range users {
		response.Users = append(response.Users, toProto(user))
	}
	return response, nil
}

func (s *Server) VerifyCredentials(ctx context.Context, req *v1.VerifyCredentialsRequest) (*v1.VerifyCredentialsResponse, error) {
	user, err := s.service.VerifyCredentials(ctx, caller(ctx), req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &v1.VerifyCredentialsResponse{User: toProto(user)}, nil
}

func toProto(u domain.User) *v1.User {
	return &v1.User{Id: u.ID, Username: u.Username, Role: u.Role, CreatedAt: timestamppb.New(u.CreatedAt)}
}
