package servicekit

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Identity is the authenticated caller attached to a request context.
type Identity struct {
	ID        string
	Username  string
	Role      string
	SessionID string
}

// Claims is the HS256 token body shared by the monolith, the Auth service and internal service tokens.
// The JSON names (sub, username, role, exp) match tokens the monolith already issues.
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type identityContextKey struct{}

// AuthUnaryInterceptor validates HS256 bearer tokens for internal RPCs.
// Health probes and the public Auth Login/RefreshToken methods remain reachable.
func AuthUnaryInterceptor(secret []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		token := BearerFromIncoming(ctx)
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "bearer token required")
		}
		identity, err := VerifyToken(token, secret)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired bearer token")
		}
		return handler(WithIdentity(ctx, identity), req)
	}
}

func isPublicMethod(method string) bool {
	// Full method names look like /drukleague.auth.v1.AuthService/Login.
	return strings.HasPrefix(method, "/grpc.health.v1.Health/") ||
		strings.HasSuffix(method, ".AuthService/Login") ||
		strings.HasSuffix(method, ".AuthService/RefreshToken")
}

// BearerFromIncoming returns the raw bearer token of an incoming gRPC request, or "".
func BearerFromIncoming(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 || !strings.HasPrefix(values[0], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer "))
}

// WithIdentity stores an identity in the context (used by the interceptor and by tests).
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok
}

func RequireRole(ctx context.Context, roles ...string) error {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authenticated identity required")
	}
	for _, role := range roles {
		if identity.Role == role {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "role is not allowed to perform this action")
}

func RequireSelfOrAdmin(ctx context.Context, userID string) error {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "authenticated identity required")
	}
	if identity.Role == RoleSuperAdmin || identity.ID == userID {
		return nil
	}
	return status.Error(codes.PermissionDenied, "cannot access another user account")
}

// SignToken issues an HS256 token with the shared claim shape.
func SignToken(secret []byte, identity Identity, ttl time.Duration) (string, time.Time, error) {
	if len(secret) == 0 || identity.ID == "" || ttl <= 0 {
		return "", time.Time{}, errors.New("signing secret, subject, and positive ttl are required")
	}
	now := time.Now()
	expires := now.Add(ttl)
	claims := Claims{
		Username: identity.Username,
		Role:     identity.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   identity.ID,
			ID:        identity.SessionID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expires, nil
}

// VerifyToken checks the signature, algorithm and expiry of an HS256 token.
func VerifyToken(token string, secret []byte) (Identity, error) {
	if len(secret) == 0 {
		return Identity{}, errors.New("signing secret is required")
	}
	claims := new(Claims)
	parsed, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		return Identity{}, errors.New("invalid or expired token")
	}
	if claims.Subject == "" {
		return Identity{}, errors.New("token subject is required")
	}
	return Identity{ID: claims.Subject, Username: claims.Username, Role: claims.Role, SessionID: claims.ID}, nil
}
