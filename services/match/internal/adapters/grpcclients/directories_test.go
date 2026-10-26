package grpcclients

import (
	"context"
	"errors"
	"net"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/match/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type teamServer struct {
	v1.UnimplementedTeamServiceServer
}

func (teamServer) GetTeam(_ context.Context, req *v1.GetTeamRequest) (*v1.GetTeamResponse, error) {
	switch req.GetId() {
	case "missing":
		return nil, status.Error(codes.NotFound, "team not found")
	case "down":
		return nil, status.Error(codes.Unavailable, "down")
	case "empty":
		return &v1.GetTeamResponse{}, nil
	}
	return &v1.GetTeamResponse{Team: &v1.Team{Id: req.GetId(), TournamentId: "t1"}}, nil
}

type userServer struct {
	v1.UnimplementedUserServiceServer
}

func (userServer) GetUser(_ context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	switch req.GetId() {
	case "missing":
		return nil, status.Error(codes.InvalidArgument, "bad id")
	case "empty":
		return &v1.GetUserResponse{}, nil
	}
	return &v1.GetUserResponse{User: &v1.User{Id: req.GetId(), Role: "match_official"}}, nil
}

func dial(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterTeamServiceServer(server, teamServer{})
	v1.RegisterUserServiceServer(server, userServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestTeamDirectory(t *testing.T) {
	teams := NewTeamDirectory(dial(t))
	if tournament, err := teams.TournamentForTeam(context.Background(), "team-1"); err != nil || tournament != "t1" {
		t.Fatalf("lookup: %v %q", err, tournament)
	}
	for id, want := range map[string]error{"missing": domain.ErrInvalid, "down": domain.ErrDependency, "empty": domain.ErrDependency} {
		if _, err := teams.TournamentForTeam(context.Background(), id); !errors.Is(err, want) {
			t.Fatalf("%s: got %v want %v", id, err, want)
		}
	}
}

func TestUserDirectory(t *testing.T) {
	users := NewUserDirectory(dial(t))
	if role, err := users.RoleForUser(context.Background(), "u1"); err != nil || role != "match_official" {
		t.Fatalf("lookup: %v %q", err, role)
	}
	for id, want := range map[string]error{"missing": domain.ErrInvalid, "empty": domain.ErrDependency} {
		if _, err := users.RoleForUser(context.Background(), id); !errors.Is(err, want) {
			t.Fatalf("%s: got %v want %v", id, err, want)
		}
	}
}
