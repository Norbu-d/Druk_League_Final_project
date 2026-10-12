package servicekit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Identity struct {
	ID       string
	Username string
	Role     string
}

type tokenClaims struct {
	ID       string `json:"sub"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Expires  int64  `json:"exp"`
}

type identityContextKey struct{}

// AuthUnaryInterceptor validates the API's HS256 bearer tokens for internal RPCs.
// Health probes and the public Auth Login/RefreshToken methods remain reachable.
func AuthUnaryInterceptor(secret []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") || strings.HasSuffix(info.FullMethod, "/AuthService/Login") || strings.HasSuffix(info.FullMethod, "/AuthService/RefreshToken") {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "bearer token required")
		}
		values := md.Get("authorization")
		if len(values) == 0 || !strings.HasPrefix(values[0], "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "bearer token required")
		}
		claims, err := verifyBearer(strings.TrimSpace(strings.TrimPrefix(values[0], "Bearer ")), secret)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired bearer token")
		}
		identity := Identity{ID: claims.ID, Username: claims.Username, Role: claims.Role}
		return handler(context.WithValue(ctx, identityContextKey{}, identity), req)
	}
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
	if identity.Role == "super_admin" || identity.ID == userID {
		return nil
	}
	return status.Error(codes.PermissionDenied, "cannot access another user account")
}

func verifyBearer(token string, secret []byte) (tokenClaims, error) {
	var claims tokenClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(secret) == 0 {
		return claims, errors.New("invalid token")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, err
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	if json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" {
		return claims, errors.New("unsupported token algorithm")
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return claims, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, &claims) != nil || claims.ID == "" || claims.Expires < time.Now().Unix() {
		return claims, errors.New("invalid or expired token")
	}
	return claims, nil
}
