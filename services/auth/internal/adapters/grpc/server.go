package grpcadapter

import (
	"context"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/auth/internal/app"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server exposes Login and RefreshToken without a bearer token (see servicekit's public
// methods); RevokeSession requires one.
type Server struct {
	authv1.UnimplementedAuthServiceServer
	service *app.Service
}

func New(service *app.Service) *Server { return &Server{service: service} }

func (s *Server) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tokens, err := s.service.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &authv1.LoginResponse{AccessToken: tokens.AccessToken, ExpiresAt: timestamppb.New(tokens.AccessExpiresAt), RefreshToken: tokens.RefreshToken, SessionId: tokens.SessionID,
		User: &v1.User{Id: tokens.User.ID, Username: tokens.User.Username, Role: tokens.User.Role}}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokens, err := s.service.Refresh(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &authv1.RefreshTokenResponse{AccessToken: tokens.AccessToken, ExpiresAt: timestamppb.New(tokens.AccessExpiresAt), RefreshToken: tokens.RefreshToken}, nil
}

func (s *Server) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.RevokeSessionResponse, error) {
	identity, _ := servicekit.IdentityFromContext(ctx)
	if err := s.service.Revoke(ctx, app.Caller{ID: identity.ID, Role: identity.Role, SessionID: identity.SessionID}, req.GetTokenId()); err != nil {
		return nil, servicekit.ToStatus(err)
	}
	return &authv1.RevokeSessionResponse{Revoked: true}, nil
}
