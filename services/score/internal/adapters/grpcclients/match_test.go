package grpcclients

import (
	"context"
	"errors"
	"net"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/services/score/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type matchServer struct {
	v1.UnimplementedMatchServiceServer
}

func (matchServer) GetMatch(_ context.Context, req *v1.GetMatchRequest) (*v1.GetMatchResponse, error) {
	switch req.GetId() {
	case "missing":
		return nil, status.Error(codes.NotFound, "no")
	case "bad":
		return nil, status.Error(codes.InvalidArgument, "no")
	case "down":
		return nil, status.Error(codes.Unavailable, "no")
	case "empty":
		return &v1.GetMatchResponse{}, nil
	}
	return &v1.GetMatchResponse{Match: &v1.Match{Id: req.GetId(), OfficialId: "o1", Status: "scheduled"}}, nil
}

func TestMatchReader(t *testing.T) {
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
	reader := NewMatchReader(conn)
	if m, err := reader.GetMatch(context.Background(), "m1"); err != nil || m.OfficialID != "o1" {
		t.Fatalf("get: %+v %v", m, err)
	}
	for id, want := range map[string]error{"missing": domain.ErrNotFound, "bad": domain.ErrInvalid, "down": domain.ErrDependency, "empty": domain.ErrDependency} {
		if _, err := reader.GetMatch(context.Background(), id); !errors.Is(err, want) {
			t.Fatalf("%s: got %v want %v", id, err, want)
		}
	}
}
