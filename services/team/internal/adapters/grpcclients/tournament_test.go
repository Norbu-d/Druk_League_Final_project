package grpcclients

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/grpctest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestTournamentsClient(t *testing.T) {
	conn := grpctest.NewClientConn(t, func(server grpc.ServiceRegistrar) { v1.RegisterTournamentServiceServer(server, tournamentServer{}) })
	tournaments := NewTournaments(conn)
	if tournament, err := tournaments.Get(context.Background(), "x1"); err != nil || tournament.ID != "x1" || tournament.Status != "registration_open" {
		t.Fatalf("get: %+v %v", tournament, err)
	}
	for id, want := range map[string]error{"missing": core.ErrNotFound, "down": core.ErrDependency, "empty": core.ErrDependency} {
		if _, err := tournaments.Get(context.Background(), id); !errors.Is(err, want) {
			t.Fatalf("%s: got %v", id, err)
		}
	}
}
