package servicekit

import (
	"errors"
	"sync"
	"time"
)

// NewServiceToken signs a short-lived bearer token for an internal service call.
func NewServiceToken(service string, secret []byte, ttl time.Duration) (string, error) {
	if service == "" {
		return "", errors.New("service name, signing secret, and positive ttl are required")
	}
	token, _, err := SignToken(secret, Identity{ID: "service:" + service, Username: service, Role: RoleService}, ttl)
	return token, err
}

// TokenSource hands out a cached service token and re-signs it before it expires,
// so long-running consumers never call other services with an expired token.
type TokenSource struct {
	service string
	secret  []byte
	ttl     time.Duration
	now     func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

func NewTokenSource(service string, secret []byte, ttl time.Duration) *TokenSource {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &TokenSource{service: service, secret: secret, ttl: ttl, now: time.Now}
}

// Token returns a token that is valid for at least a fifth of its lifetime.
func (s *TokenSource) Token() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && s.now().Add(s.ttl/5).Before(s.expires) {
		return s.token, nil
	}
	token, err := NewServiceToken(s.service, s.secret, s.ttl)
	if err != nil {
		return "", err
	}
	s.token, s.expires = token, s.now().Add(s.ttl)
	return token, nil
}
