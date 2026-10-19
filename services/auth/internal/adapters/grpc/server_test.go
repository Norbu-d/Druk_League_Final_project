package grpcadapter

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/auth/internal/app"
	"github.com/drukleague/platform/services/auth/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const sessionID = "22222222-2222-4222-8222-222222222222"

type users struct{}

func (users) Verify(_ context.Context, u, p string) (domain.User, error) {
	if p != "correct-password" {
		return domain.User{}, core.ErrUnauthenticated
	}
	return domain.User{ID: "u1", Username: u, Role: core.RoleViewer}, nil
}
func (users) Get(context.Context, string) (domain.User, error) {
	return domain.User{ID: "u1", Role: core.RoleViewer}, nil
}

type sessions struct{ hash string }

func (s *sessions) Create(_ context.Context, user, hash string, exp time.Time) (domain.Session, error) {
	s.hash = hash
	return domain.Session{ID: sessionID, UserID: user, ExpiresAt: exp}, nil
}
func (s *sessions) ByRefreshHash(_ context.Context, hash string) (domain.Session, error) {
	if hash != s.hash {
		return domain.Session{}, core.NotFound("session")
	}
	return domain.Session{ID: sessionID, UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (s *sessions) Rotate(_ context.Context, _, _, newHash string, _ time.Time) (bool, error) {
	s.hash = newHash
	return true, nil
}
func (s *sessions) Get(context.Context, string) (domain.Session, error) {
	return domain.Session{ID: sessionID, UserID: "u1"}, nil
}
func (s *sessions) Revoke(context.Context, string) error { return nil }

type attempts struct{}

func (attempts) Get(context.Context, string) (domain.Attempts, error) { return domain.Attempts{}, nil }
func (attempts) Save(context.Context, string, domain.Attempts) error  { return nil }
func (attempts) Reset(context.Context, string) error                  { return nil }

type signer struct{}

func (signer) Sign(u domain.User, s string, ttl time.Duration) (string, time.Time, error) {
	return "jwt", time.Now().Add(ttl), nil
}

func TestAuthRPCs(t *testing.T) {
	s := New(app.New(users{}, &sessions{}, attempts{}, signer{}, app.Config{}))
	login, err := s.Login(context.Background(), &authv1.LoginRequest{Username: "admin", Password: "correct-password"})
	if err != nil || login.GetAccessToken() != "jwt" || login.GetRefreshToken() == "" || login.GetSessionId() != sessionID || login.GetUser().GetRole() != core.RoleViewer {
		t.Fatalf("login: %v %+v", err, login)
	}
	if _, err := s.Login(context.Background(), &authv1.LoginRequest{Username: "admin", Password: "wrong"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("wrong password: %v", err)
	}
	refreshed, err := s.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: login.GetRefreshToken()})
	if err != nil || refreshed.GetRefreshToken() == login.GetRefreshToken() || refreshed.GetExpiresAt() == nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := s.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: "bogus"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("bogus refresh: %v", err)
	}
	owner := servicekit.WithIdentity(context.Background(), servicekit.Identity{ID: "u1", Role: core.RoleViewer, SessionID: sessionID})
	if revoked, err := s.RevokeSession(owner, &authv1.RevokeSessionRequest{}); err != nil || !revoked.GetRevoked() {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := s.RevokeSession(context.Background(), &authv1.RevokeSessionRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("anonymous revoke: %v", err)
	}
}
