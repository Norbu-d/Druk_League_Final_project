package grpcclients

import (
	"context"
	"errors"
	"net"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
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
		return nil, status.Error(codes.NotFound, "no")
	case "down":
		return nil, status.Error(codes.Unavailable, "no")
	case "empty":
		return &v1.GetTeamResponse{}, nil
	}
	return &v1.GetTeamResponse{Team: &v1.Team{Id: req.GetId(), ManagerId: "m1"}}, nil
}

func connect(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterTeamServiceServer(server, teamServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestTeamsClient(t *testing.T) {
	teams := NewTeams(connect(t))
	if team, err := teams.Get(context.Background(), "t1"); err != nil || team.ID != "t1" || team.ManagerID != "m1" {
		t.Fatalf("get: %+v %v", team, err)
	}
	for id, want := range map[string]error{"missing": core.ErrNotFound, "down": core.ErrDependency, "empty": core.ErrDependency} {
		if _, err := teams.Get(context.Background(), id); !errors.Is(err, want) {
			t.Fatalf("%s: got %v", id, err)
		}
	}
}
