package servicekit

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestBreakerLifecycle(t *testing.T) {
	now := time.Now()
	breaker := NewCircuitBreaker(2, time.Second)
	breaker.now = func() time.Time { return now }

	if !breaker.Allow() || breaker.State() != BreakerClosed {
		t.Fatal("new breaker must be closed")
	}
	breaker.Record(true)
	if breaker.State() != BreakerClosed {
		t.Fatal("one failure must not open the breaker")
	}
	breaker.Record(true)
	if breaker.State() != BreakerOpen || breaker.Allow() {
		t.Fatal("breaker must open after threshold and reject calls")
	}
	now = now.Add(time.Second)
	if !breaker.Allow() || breaker.State() != BreakerHalfOpen {
		t.Fatal("breaker must allow one probe after cool-down")
	}
	if breaker.Allow() {
		t.Fatal("only one probe is allowed while half-open")
	}
	breaker.Record(true)
	if breaker.State() != BreakerOpen {
		t.Fatal("failed probe must reopen the breaker")
	}
	now = now.Add(time.Second)
	breaker.Allow()
	breaker.Record(false)
	if breaker.State() != BreakerClosed || !breaker.Allow() {
		t.Fatal("successful probe must close the breaker")
	}
}

func TestBreakerSuccessResetsFailures(t *testing.T) {
	breaker := NewCircuitBreaker(2, time.Second)
	breaker.Record(true)
	breaker.Record(false)
	breaker.Record(true)
	if breaker.State() != BreakerClosed {
		t.Fatal("failures must be consecutive to open the breaker")
	}
}

func TestBreakerDefaultsAndStateNames(t *testing.T) {
	breaker := NewCircuitBreaker(0, 0)
	if breaker.threshold != 5 || breaker.coolDown != 10*time.Second {
		t.Fatal("defaults not applied")
	}
	for state, name := range map[BreakerState]string{BreakerClosed: "closed", BreakerOpen: "open", BreakerHalfOpen: "half-open"} {
		if state.String() != name {
			t.Fatalf("%d should be %s", state, name)
		}
	}
}

func TestDependencyFailureClassification(t *testing.T) {
	for code, want := range map[codes.Code]bool{codes.Unavailable: true, codes.DeadlineExceeded: true, codes.Internal: true, codes.NotFound: false, codes.PermissionDenied: false, codes.InvalidArgument: false} {
		if isDependencyFailure(status.Error(code, "x")) != want {
			t.Fatalf("%s: want %v", code, want)
		}
	}
	if isDependencyFailure(nil) {
		t.Fatal("nil is not a failure")
	}
}

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := RecoveryUnaryInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x/Y"}, func(context.Context, any) (any, error) { panic("boom") })
	if status.Code(err) != codes.Internal {
		t.Fatalf("panic must become Internal, got %v", err)
	}
	response, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x/Y"}, func(context.Context, any) (any, error) { return "ok", nil })
	if err != nil || response != "ok" {
		t.Fatal("normal handler must pass through")
	}
}

func TestLoggingInterceptorAssignsOrPropagatesRequestID(t *testing.T) {
	interceptor := LoggingUnaryInterceptor("test-service")
	var seen string
	handler := func(ctx context.Context, _ any) (any, error) { seen = RequestID(ctx); return nil, nil }
	_, _ = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x/Y"}, handler)
	if len(seen) != 16 {
		t.Fatalf("expected generated request id, got %q", seen)
	}
	incoming := metadata.NewIncomingContext(context.Background(), metadata.Pairs(requestIDHeader, "abc"))
	_, _ = interceptor(incoming, nil, &grpc.UnaryServerInfo{FullMethod: "/x/Y"}, handler)
	if seen != "abc" {
		t.Fatalf("expected propagated request id, got %q", seen)
	}
	seen = "unchanged"
	_, _ = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}, handler)
	if seen != "" {
		t.Fatal("health checks skip request ids")
	}
}

func TestHelpers(t *testing.T) {
	if size, offset, err := Page(0, ""); err != nil || size != 50 || offset != 0 {
		t.Fatal("default page wrong")
	}
	if size, offset, err := Page(500, "20"); err != nil || size != 100 || offset != 20 {
		t.Fatal("page clamp wrong")
	}
	if _, _, err := Page(10, "-1"); status.Code(err) != codes.InvalidArgument {
		t.Fatal("negative token accepted")
	}
	if _, _, err := Page(10, "abc"); status.Code(err) != codes.InvalidArgument {
		t.Fatal("non-numeric token accepted")
	}
	if NextPageToken(10, 5, false) != "" || NextPageToken(10, 5, true) != "15" {
		t.Fatal("next page token wrong")
	}
	id, err := NewUUID()
	if err != nil || !IsUUID(id) {
		t.Fatalf("bad uuid %q", id)
	}
	if IsUUID("not-a-uuid") || IsUUID("") {
		t.Fatal("invalid uuid accepted")
	}
	t.Setenv("SERVICEKIT_TEST_VALUE", "set")
	if Env("SERVICEKIT_TEST_VALUE", "x") != "set" || Env("SERVICEKIT_TEST_MISSING", "fallback") != "fallback" {
		t.Fatal("env helper wrong")
	}
	t.Setenv("JWT_SECRET", "")
	if string(Secret()) != developmentSecret {
		t.Fatal("development secret fallback wrong")
	}
	if len(NewRequestID()) != 16 {
		t.Fatal("request id length wrong")
	}
}
