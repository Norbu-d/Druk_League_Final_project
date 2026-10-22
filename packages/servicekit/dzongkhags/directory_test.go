package dzongkhags

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	dzongkhagv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/dzongkhag/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type server struct {
	dzongkhagv1.UnimplementedDzongkhagServiceServer
	down  atomic.Bool
	calls atomic.Int32
}

func (s *server) ListDzongkhags(context.Context, *dzongkhagv1.ListDzongkhagsRequest) (*dzongkhagv1.ListDzongkhagsResponse, error) {
	s.calls.Add(1)
	if s.down.Load() {
		return nil, status.Error(codes.Unavailable, "down")
	}
	return &dzongkhagv1.ListDzongkhagsResponse{Dzongkhags: []*dzongkhagv1.Dzongkhag{{Code: "14", Name: "Thimphu"}, {Code: "19", Name: "Wangdue Phodrang"}}}, nil
}

func start(t *testing.T) (*server, *grpc.ClientConn) {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	grpcServer := grpc.NewServer()
	fake := &server{}
	dzongkhagv1.RegisterDzongkhagServiceServer(grpcServer, fake)
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(grpcServer.Stop)
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return fake, conn
}

func TestCanonicalNamesAndCodes(t *testing.T) {
	fake, conn := start(t)
	directory := New(conn, time.Hour)
	for input, want := range map[string]string{"thimphu": "Thimphu", " WANGDUE PHODRANG ": "Wangdue Phodrang", "14": "Thimphu"} {
		if got, err := directory.Canonical(context.Background(), input); err != nil || got != want {
			t.Fatalf("%q: got %q %v", input, got, err)
		}
	}
	if fake.calls.Load() != 1 {
		t.Fatalf("directory should be cached, got %d calls", fake.calls.Load())
	}
	if _, err := directory.Canonical(context.Background(), "Atlantis"); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("unknown district: %v", err)
	}
	if _, err := directory.Canonical(context.Background(), " "); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("empty district: %v", err)
	}
}

func TestStaleCacheDuringOutage(t *testing.T) {
	fake, conn := start(t)
	directory := New(conn, time.Minute)
	now := time.Now()
	directory.now = func() time.Time { return now }
	if _, err := directory.Canonical(context.Background(), "Thimphu"); err != nil {
		t.Fatal(err)
	}
	fake.down.Store(true)
	now = now.Add(time.Hour)
	if got, err := directory.Canonical(context.Background(), "Thimphu"); err != nil || got != "Thimphu" {
		t.Fatalf("stale cache not used: %q %v", got, err)
	}
}

func TestOutageWithoutCache(t *testing.T) {
	fake, conn := start(t)
	fake.down.Store(true)
	if _, err := New(conn, 0).Canonical(context.Background(), "Thimphu"); !errors.Is(err, core.ErrDependency) {
		t.Fatalf("expected dependency error, got %v", err)
	}
}
