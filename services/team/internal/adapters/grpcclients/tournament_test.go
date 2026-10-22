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

type tournamentServer struct {
	v1.UnimplementedTournamentServiceServer
}

func (tournamentServer) GetTournament(_ context.Context, req *v1.GetTournamentRequest) (*v1.GetTournamentResponse, error) {
	switch req.GetId() {
	case "missing":
		return nil, status.Error(codes.NotFound, "no")
	case "down":
		return nil, status.Error(codes.Unavailable, "no")
	case "empty":
		return &v1.GetTournamentResponse{}, nil
	}
	return &v1.GetTournamentResponse{Tournament: &v1.Tournament{Id: req.GetId(), Status: "registration_open"}}, nil
}

func connect(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterTournamentServiceServer(server, tournamentServer{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestTournamentsClient(t *testing.T) {
	tournaments := NewTournaments(connect(t))
	if tournament, err := tournaments.Get(context.Background(), "x1"); err != nil || tournament.ID != "x1" || tournament.Status != "registration_open" {
		t.Fatalf("get: %+v %v", tournament, err)
	}
	for id, want := range map[string]error{"missing": core.ErrNotFound, "down": core.ErrDependency, "empty": core.ErrDependency} {
		if _, err := tournaments.Get(context.Background(), id); !errors.Is(err, want) {
			t.Fatalf("%s: got %v", id, err)
		}
	}
}
