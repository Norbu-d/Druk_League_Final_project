package servicekit

import (
	"os"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const developmentSecret = "local-development-secret-change-before-deployment"

// Env returns the environment variable or a fallback.
func Env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// Secret returns the shared JWT signing secret.
func Secret() []byte { return []byte(Env("JWT_SECRET", developmentSecret)) }

// Page validates the shared offset pagination: size 1..100 (default 50) and a numeric token.
func Page(size int32, token string) (int, int, error) {
	if size <= 0 {
		size = 50
	}
	if size > 100 {
		size = 100
	}
	offset := 0
	if token != "" {
		n, err := strconv.Atoi(token)
		if err != nil || n < 0 {
			return 0, 0, status.Error(codes.InvalidArgument, "invalid page token")
		}
		offset = n
	}
	return int(size), offset, nil
}

// NextPageToken returns the token for the next page, or "" when there is none.
func NextPageToken(offset, returned int, more bool) string {
	if !more {
		return ""
	}
	return strconv.Itoa(offset + returned)
}
