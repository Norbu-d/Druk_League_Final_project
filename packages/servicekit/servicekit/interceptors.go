package servicekit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requestIDHeader = "x-request-id"

type requestIDKey struct{}

// RecoveryUnaryInterceptor turns a handler panic into codes.Internal instead of crashing the service.
func RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("rpc panic method=%s: %v\n%s", info.FullMethod, recovered, debug.Stack())
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// LoggingUnaryInterceptor attaches a request id (propagated or new) and logs one line per RPC.
func LoggingUnaryInterceptor(service string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}
		id := incomingRequestID(ctx)
		if id == "" {
			id = NewRequestID()
		}
		ctx = WithRequestID(ctx, id)
		start := time.Now()
		response, err := handler(ctx, req)
		log.Printf("rpc service=%s method=%s code=%s duration=%s request_id=%s", service, info.FullMethod, status.Code(err), time.Since(start).Round(time.Microsecond), id)
		return response, err
	}
}

func incomingRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(requestIDHeader); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// WithRequestID stores the request id that outbound calls propagate.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the request id of the current call, if any.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// NewRequestID returns 16 random hex characters.
func NewRequestID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(raw[:])
}
