package grpcclients

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"google.golang.org/grpc"
)

type fakeConn struct {
	response *v1.GetTournamentResponse
	err      error
	method   string
	request  *v1.GetTournamentRequest
}

func (c *fakeConn) Invoke(_ context.Context, method string, args, reply any, _ ...grpc.CallOption) error {
	c.method = method
	c.request = args.(*v1.GetTournamentRequest)
	if c.err != nil {
		return c.err
	}
	*reply.(*v1.GetTournamentResponse) = *c.response
	return nil
}
func (*fakeConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, errors.New("unexpected stream")
}

func TestGetTournament(t *testing.T) {
	conn := &fakeConn{response: &v1.GetTournamentResponse{Tournament: &v1.Tournament{Id: "id", Season: "2026", Dzongkhag: "Paro"}}}
	got, err := NewTournaments(conn).GetTournament(context.Background(), "requested")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "id" || got.Season != "2026" || got.Dzongkhag != "Paro" || conn.request.GetId() != "requested" {
		t.Fatalf("tournament=%+v request=%+v", got, conn.request)
	}
	conn.response = &v1.GetTournamentResponse{}
	if _, err := NewTournaments(conn).GetTournament(context.Background(), "id"); err == nil {
		t.Fatal("expected missing tournament error")
	}
	conn.err = errors.New("rpc failed")
	if _, err := NewTournaments(conn).GetTournament(context.Background(), "id"); err == nil {
		t.Fatal("expected RPC error")
	}
}
