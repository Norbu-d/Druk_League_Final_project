package servicekit

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeInvoker struct {
	calls   int
	errs    []error
	lastCtx context.Context
}

func (f *fakeInvoker) invoke(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
	f.calls++
	f.lastCtx = ctx
	if len(f.errs) == 0 {
		return nil
	}
	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}

func fastOptions() ClientOptions {
	return ClientOptions{Timeout: time.Second, MaxAttempts: 3, BaseBackoff: time.Millisecond, BreakerThreshold: 3, BreakerCoolDown: time.Hour}
}

func TestSafeReadsAreRetriedOnUnavailable(t *testing.T) {
	interceptor, _ := NewClientInterceptor("team:50051", fastOptions())
	invoker := &fakeInvoker{errs: []error{status.Error(codes.Unavailable, "down"), status.Error(codes.DeadlineExceeded, "slow")}}
	if err := interceptor(context.Background(), "/drukleague.v1.TeamService/GetTeam", nil, nil, nil, invoker.invoke); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if invoker.calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", invoker.calls)
	}
}

func TestWritesAndBusinessErrorsAreNotRetried(t *testing.T) {
	interceptor, _ := NewClientInterceptor("team:50051", fastOptions())
	invoker := &fakeInvoker{errs: []error{status.Error(codes.Unavailable, "down")}}
	if err := interceptor(context.Background(), "/drukleague.v1.TeamService/CreateTeam", nil, nil, nil, invoker.invoke); status.Code(err) != codes.Unavailable || invoker.calls != 1 {
		t.Fatalf("write retried: calls=%d err=%v", invoker.calls, err)
	}
	invoker = &fakeInvoker{errs: []error{status.Error(codes.NotFound, "missing")}}
	if err := interceptor(context.Background(), "/drukleague.v1.TeamService/GetTeam", nil, nil, nil, invoker.invoke); status.Code(err) != codes.NotFound || invoker.calls != 1 {
		t.Fatalf("business error retried: calls=%d err=%v", invoker.calls, err)
	}
}

func TestBreakerOpensAndRejectsWithoutCallingDependency(t *testing.T) {
	options := fastOptions()
	options.MaxAttempts = 1
	interceptor, breaker := NewClientInterceptor("team:50051", options)
	down := status.Error(codes.Unavailable, "down")
	invoker := &fakeInvoker{errs: []error{down, down, down}}
	for i := 0; i < 3; i++ {
		_ = interceptor(context.Background(), "/drukleague.v1.TeamService/GetTeam", nil, nil, nil, invoker.invoke)
	}
	if breaker.State() != BreakerOpen {
		t.Fatalf("breaker should be open, is %s", breaker.State())
	}
	err := interceptor(context.Background(), "/drukleague.v1.TeamService/GetTeam", nil, nil, nil, invoker.invoke)
	if !errors.Is(err, errCircuitOpen) || invoker.calls != 3 {
		t.Fatalf("open breaker must fail fast: calls=%d err=%v", invoker.calls, err)
	}
	if status.Code(err) != codes.Unavailable {
		t.Fatal("open breaker must surface Unavailable")
	}
}

func TestDeadlineIsApplied(t *testing.T) {
	options := fastOptions()
	options.Timeout = 50 * time.Millisecond
	interceptor, _ := NewClientInterceptor("x", options)
	invoker := &fakeInvoker{}
	_ = interceptor(context.Background(), "/svc/CreateThing", nil, nil, nil, invoker.invoke)
	deadline, ok := invoker.lastCtx.Deadline()
	if !ok || time.Until(deadline) > 50*time.Millisecond {
		t.Fatalf("expected per-attempt deadline, got %v %v", deadline, ok)
	}
}

func TestAuthModes(t *testing.T) {
	source := NewTokenSource("bff", testSecret, time.Minute)
	callerCtx := WithRequestID(WithBearer(context.Background(), "caller-token"), "req-1")

	check := func(mode AuthMode, ctx context.Context, wantCaller, wantService bool) {
		t.Helper()
		options := fastOptions()
		options.Auth, options.Tokens = mode, source
		interceptor, _ := NewClientInterceptor("x", options)
		invoker := &fakeInvoker{}
		if err := interceptor(ctx, "/svc/GetThing", nil, nil, nil, invoker.invoke); err != nil {
			t.Fatal(err)
		}
		md, _ := metadata.FromOutgoingContext(invoker.lastCtx)
		auth := md.Get("authorization")
		gotCaller := len(auth) == 1 && auth[0] == "Bearer caller-token"
		gotService := len(auth) == 1 && auth[0] != "Bearer caller-token"
		if gotCaller != wantCaller || gotService != wantService {
			t.Fatalf("mode %d: authorization=%v", mode, auth)
		}
		if RequestID(ctx) != "" && (len(md.Get(requestIDHeader)) != 1 || md.Get(requestIDHeader)[0] != "req-1") {
			t.Fatalf("request id not propagated: %v", md)
		}
	}
	check(ForwardCaller, callerCtx, true, false)
	check(ForwardCaller, context.Background(), false, false)
	check(ServiceIdentity, callerCtx, false, true)
	check(ForwardOrService, callerCtx, true, false)
	check(ForwardOrService, context.Background(), false, true)
}

func TestFailingTokenSourceIsUnauthenticated(t *testing.T) {
	options := fastOptions()
	options.Auth, options.Tokens = ServiceIdentity, NewTokenSource("", testSecret, time.Minute)
	interceptor, _ := NewClientInterceptor("x", options)
	invoker := &fakeInvoker{}
	if err := interceptor(context.Background(), "/svc/GetThing", nil, nil, nil, invoker.invoke); status.Code(err) != codes.Unauthenticated || invoker.calls != 0 {
		t.Fatalf("expected Unauthenticated without a call, got %v (%d calls)", err, invoker.calls)
	}
}

func TestCancelledContextStopsRetries(t *testing.T) {
	options := fastOptions()
	options.BaseBackoff = time.Hour
	interceptor, _ := NewClientInterceptor("x", options)
	ctx, cancel := context.WithCancel(context.Background())
	invoker := &fakeInvoker{errs: []error{status.Error(codes.Unavailable, "down")}}
	go func() { time.Sleep(10 * time.Millisecond); cancel() }()
	if err := interceptor(ctx, "/svc/GetThing", nil, nil, nil, invoker.invoke); status.Code(err) != codes.Unavailable || invoker.calls != 1 {
		t.Fatalf("expected one attempt then stop, got %d calls err=%v", invoker.calls, err)
	}
}

func TestIsSafeMethod(t *testing.T) {
	for method, want := range map[string]bool{
		"/a.B/GetTeam": true, "/a.B/ListTeams": true, "/a.B/Search": true, "/a.B/VerifyCredentials": true,
		"/a.B/CreateTeam": false, "/a.B/FinalizeMatch": false, "/a.B/AssignOfficial": false,
	} {
		if IsSafeMethod(method) != want {
			t.Fatalf("%s: want %v", method, want)
		}
	}
}

func TestDialBuildsConnection(t *testing.T) {
	conn, err := Dial("localhost:1", ClientOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
}
