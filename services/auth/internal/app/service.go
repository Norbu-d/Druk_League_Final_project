package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/auth/internal/domain"
	"github.com/drukleague/platform/services/auth/internal/ports"
)

type Caller struct{ ID, Role, SessionID string }

type Config struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Lockout    domain.LockoutPolicy
}

type Service struct {
	users    ports.Users
	sessions ports.Sessions
	attempts ports.Attempts
	signer   ports.Signer
	config   Config
	now      func() time.Time
	random   func() (string, error)
}

func New(users ports.Users, sessions ports.Sessions, attempts ports.Attempts, signer ports.Signer, config Config) *Service {
	if config.AccessTTL <= 0 {
		config.AccessTTL = 30 * time.Minute
	}
	if config.RefreshTTL <= 0 {
		config.RefreshTTL = 7 * 24 * time.Hour
	}
	if config.Lockout.MaxFailures <= 0 {
		config.Lockout = domain.DefaultLockout
	}
	return &Service{users: users, sessions: sessions, attempts: attempts, signer: signer, config: config, now: time.Now, random: randomToken}
}

var errBadCredentials = fmt.Errorf("%w: invalid username or password", core.ErrUnauthenticated)

// Login verifies credentials with the User service and opens a session.
func (s *Service) Login(ctx context.Context, username, password string) (domain.Tokens, error) {
	key := strings.ToLower(strings.TrimSpace(username))
	if key == "" || password == "" {
		return domain.Tokens{}, errBadCredentials
	}
	attempts, err := s.attempts.Get(ctx, key)
	if err != nil {
		return domain.Tokens{}, err
	}
	if s.config.Lockout.Locked(attempts, s.now()) {
		return domain.Tokens{}, core.Precondition("too many failed sign-in attempts; try again in 15 minutes")
	}
	user, err := s.users.Verify(ctx, username, password)
	if errors.Is(err, core.ErrUnauthenticated) {
		if saveErr := s.attempts.Save(ctx, key, s.config.Lockout.AfterFailure(attempts, s.now())); saveErr != nil {
			return domain.Tokens{}, saveErr
		}
		return domain.Tokens{}, errBadCredentials
	}
	if err != nil {
		return domain.Tokens{}, err
	}
	if err := s.attempts.Reset(ctx, key); err != nil {
		return domain.Tokens{}, err
	}
	refresh, err := s.random()
	if err != nil {
		return domain.Tokens{}, err
	}
	session, err := s.sessions.Create(ctx, user.ID, hashToken(refresh), s.now().Add(s.config.RefreshTTL))
	if err != nil {
		return domain.Tokens{}, err
	}
	return s.issue(user, session.ID, refresh)
}

// Refresh exchanges a refresh token for a new access token and a new refresh token.
// The user's current role is read again, so a role change applies at the next refresh.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (domain.Tokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return domain.Tokens{}, core.ErrUnauthenticated
	}
	oldHash := hashToken(refreshToken)
	session, err := s.sessions.ByRefreshHash(ctx, oldHash)
	if errors.Is(err, core.ErrNotFound) {
		return domain.Tokens{}, core.ErrUnauthenticated
	}
	if err != nil {
		return domain.Tokens{}, err
	}
	if session.Revoked || !s.now().Before(session.ExpiresAt) {
		return domain.Tokens{}, core.ErrUnauthenticated
	}
	user, err := s.users.Get(ctx, session.UserID)
	if errors.Is(err, core.ErrNotFound) {
		_ = s.sessions.Revoke(ctx, session.ID)
		return domain.Tokens{}, core.ErrUnauthenticated
	}
	if err != nil {
		return domain.Tokens{}, err
	}
	next, err := s.random()
	if err != nil {
		return domain.Tokens{}, err
	}
	rotated, err := s.sessions.Rotate(ctx, session.ID, oldHash, hashToken(next), s.now().Add(s.config.RefreshTTL))
	if err != nil {
		return domain.Tokens{}, err
	}
	if !rotated {
		return domain.Tokens{}, core.ErrUnauthenticated
	}
	return s.issue(user, session.ID, next)
}

// Revoke signs a session out. An empty id means the caller's own session.
func (s *Service) Revoke(ctx context.Context, caller Caller, sessionID string) error {
	if caller.ID == "" {
		return core.ErrUnauthenticated
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = caller.SessionID
	}
	if !core.IsUUID(sessionID) {
		return core.Invalid("session id must be a UUID")
	}
	session, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.UserID != caller.ID && caller.Role != core.RoleSuperAdmin {
		return core.Forbidden("cannot revoke another user's session")
	}
	return s.sessions.Revoke(ctx, sessionID)
}

func (s *Service) issue(user domain.User, sessionID, refresh string) (domain.Tokens, error) {
	access, expires, err := s.signer.Sign(user, sessionID, s.config.AccessTTL)
	if err != nil {
		return domain.Tokens{}, err
	}
	return domain.Tokens{AccessToken: access, AccessExpiresAt: expires, RefreshToken: refresh, SessionID: sessionID, User: user}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
