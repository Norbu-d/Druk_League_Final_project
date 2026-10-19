package servicekit

import (
	"context"
	"errors"
	"log"
	"path"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthMode selects which bearer token an outbound call carries.
type AuthMode int

const (
	// ForwardCaller copies the caller's bearer token, so the callee applies the caller's role.
	ForwardCaller AuthMode = iota
	// ServiceIdentity always uses the service's own token (background consumers, lookups).
	ServiceIdentity
	// ForwardOrService forwards the caller's token when present and falls back to the service token.
	ForwardOrService
)

// ClientOptions configures the resilience policy of an outbound gRPC connection.
type ClientOptions struct {
	Timeout          time.Duration // per attempt; default 3s
	MaxAttempts      int           // for safe reads; default 3
	BaseBackoff      time.Duration // default 100ms, doubled per retry
	BreakerThreshold int           // consecutive failures before opening; default 5
	BreakerCoolDown  time.Duration // default 10s
	Auth             AuthMode
	Tokens           *TokenSource // required for ServiceIdentity and ForwardOrService
}

var errCircuitOpen = status.Error(codes.Unavailable, "dependency circuit is open")

// Dial opens a client connection whose calls get a deadline, retries on safe reads,
// a circuit breaker per target, request-id propagation and the selected bearer token.
func Dial(target string, options ClientOptions) (*grpc.ClientConn, error) {
	interceptor, _ := NewClientInterceptor(target, options)
	return grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(interceptor))
}

// NewClientInterceptor builds the resilience interceptor used by Dial; the breaker is returned for tests and metrics.
func NewClientInterceptor(target string, options ClientOptions) (grpc.UnaryClientInterceptor, *CircuitBreaker) {
	if options.Timeout <= 0 {
		options.Timeout = 3 * time.Second
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = 3
	}
	if options.BaseBackoff <= 0 {
		options.BaseBackoff = 100 * time.Millisecond
	}
	breaker := NewCircuitBreaker(options.BreakerThreshold, options.BreakerCoolDown)
	interceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
		ctx, err := outgoingContext(ctx, options)
		if err != nil {
			return status.Error(codes.Unauthenticated, "could not create service token")
		}
		attempts := 1
		if IsSafeMethod(method) {
			attempts = options.MaxAttempts
		}
		backoff := options.BaseBackoff
		for attempt := 1; ; attempt++ {
			err = callOnce(ctx, breaker, options.Timeout, method, req, reply, cc, invoker, callOptions...)
			if err == nil || attempt >= attempts || !retryable(err) || ctx.Err() != nil {
				if errors.Is(err, errCircuitOpen) {
					log.Printf("rpc client target=%s method=%s circuit=open", target, method)
				}
				return err
			}
			select {
			case <-ctx.Done():
				return err
			case <-time.After(backoff):
			}
			backoff *= 2
		}
	}
	return interceptor, breaker
}

func callOnce(ctx context.Context, breaker *CircuitBreaker, timeout time.Duration, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
	if !breaker.Allow() {
		return errCircuitOpen
	}
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := invoker(attemptCtx, method, req, reply, cc, callOptions...)
	breaker.Record(isDependencyFailure(err))
	return err
}

// IsSafeMethod reports whether an RPC only reads data and may be retried.
func IsSafeMethod(method string) bool {
	name := path.Base(method)
	return strings.HasPrefix(name, "Get") || strings.HasPrefix(name, "List") || strings.HasPrefix(name, "Search") || strings.HasPrefix(name, "Verify")
}

func retryable(err error) bool {
	if errors.Is(err, errCircuitOpen) {
		return false
	}
	code := status.Code(err)
	return code == codes.Unavailable || code == codes.DeadlineExceeded
}

// isDependencyFailure separates infrastructure failures (which trip the breaker)
// from business errors such as NotFound or PermissionDenied (which do not).
func isDependencyFailure(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Internal, codes.Unknown:
		return err != nil
	}
	return false
}

func outgoingContext(ctx context.Context, options ClientOptions) (context.Context, error) {
	pairs := []string{}
	if id := RequestID(ctx); id != "" {
		pairs = append(pairs, requestIDHeader, id)
	}
	caller := BearerFromIncoming(ctx)
	switch {
	case options.Auth == ForwardCaller && caller != "":
		pairs = append(pairs, "authorization", "Bearer "+caller)
	case options.Auth == ForwardOrService && caller != "":
		pairs = append(pairs, "authorization", "Bearer "+caller)
	case options.Auth != ForwardCaller && options.Tokens != nil:
		token, err := options.Tokens.Token()
		if err != nil {
			return ctx, err
		}
		pairs = append(pairs, "authorization", "Bearer "+token)
	}
	if len(pairs) == 0 {
		return ctx, nil
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...), nil
}

// WithBearer marks a context so that outbound calls forward the given bearer token.
// HTTP front ends (gateway, BFF) use it because they receive tokens over HTTP, not gRPC.
func WithBearer(ctx context.Context, token string) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	md = md.Copy()
	md.Set("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(ctx, md)
}
