package grpcclients

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/auth/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type userServer struct {
	v1.UnimplementedUserServiceServer
}

func (userServer) VerifyCredentials(_ context.Context, req *v1.VerifyCredentialsRequest) (*v1.VerifyCredentialsResponse, error) {
	switch req.GetUsername() {
	case "wrong":
		return nil, status.Error(codes.Unauthenticated, "no")
	case "down":
		return nil, status.Error(codes.Unavailable, "no")
	}
	return &v1.VerifyCredentialsResponse{User: &v1.User{Id: "u1", Username: req.GetUsername(), Role: "viewer"}}, nil
}
func (userServer) GetUser(_ context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	if req.GetId() == "gone" {
		return nil, status.Error(codes.NotFound, "no")
	}
	return &v1.GetUserResponse{User: &v1.User{Id: req.GetId(), Role: "match_official"}}, nil
}

func TestUsersClientAndSigner(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterUserServiceServer(server, userServer{})
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	users := NewUsers(conn)
	if u, err := users.Verify(context.Background(), "karma", "x"); err != nil || u.Role != "viewer" {
		t.Fatalf("verify: %+v %v", u, err)
	}
	if _, err := users.Verify(context.Background(), "wrong", "x"); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatalf("wrong: %v", err)
	}
	if _, err := users.Verify(context.Background(), "down", "x"); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("down: %v", err)
	}
	if u, err := users.Get(context.Background(), "u2"); err != nil || u.Role != "match_official" {
		t.Fatalf("get: %+v %v", u, err)
	}
	if _, err := users.Get(context.Background(), "gone"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("gone: %v", err)
	}

	secret := []byte("signer-test-secret-signer-test-secret")
	token, expires, err := NewSigner(secret).Sign(domain.User{ID: "u1", Username: "karma", Role: "viewer"}, "s1", time.Minute)
	if err != nil || time.Until(expires) <= 0 {
		t.Fatal(err)
	}
	identity, err := servicekit.VerifyToken(token, secret)
	if err != nil || identity.SessionID != "s1" || identity.Role != "viewer" {
		t.Fatalf("token identity %+v %v", identity, err)
	}
}
