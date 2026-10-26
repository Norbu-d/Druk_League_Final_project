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

type matchServer struct {
	v1.UnimplementedMatchServiceServer
}

func (matchServer) AssignOfficial(_ context.Context, req *v1.AssignOfficialRequest) (*v1.AssignOfficialResponse, error) {
	switch req.GetMatchId() {
	case "viewer":
		return nil, status.Error(codes.InvalidArgument, "no")
	case "done":
		return nil, status.Error(codes.AlreadyExists, "no")
	case "missing":
		return nil, status.Error(codes.NotFound, "no")
	case "down":
		return nil, status.Error(codes.Unavailable, "no")
	case "empty":
		return &v1.AssignOfficialResponse{}, nil
	}
	return &v1.AssignOfficialResponse{Match: &v1.Match{Id: req.GetMatchId(), TournamentId: "t1", OfficialId: req.GetOfficialId()}}, nil
}

func TestMatchesClient(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterMatchServiceServer(server, matchServer{})
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	matches := NewMatches(conn)
	if ref, err := matches.AssignOfficial(context.Background(), "m1", "o1"); err != nil || ref.TournamentID != "t1" || ref.OfficialID != "o1" {
		t.Fatalf("assign: %+v %v", ref, err)
	}
	for id, want := range map[string]error{"viewer": core.ErrInvalid, "done": core.ErrConflict, "missing": core.ErrNotFound, "down": core.ErrDependency, "empty": core.ErrDependency} {
		if _, err := matches.AssignOfficial(context.Background(), id, "o1"); !errors.Is(err, want) {
			t.Fatalf("%s: got %v", id, err)
		}
	}
}
