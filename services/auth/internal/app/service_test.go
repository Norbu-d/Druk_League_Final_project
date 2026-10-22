package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/auth/internal/domain"
)

const (
	userID    = "11111111-1111-4111-8111-111111111111"
	sessionID = "22222222-2222-4222-8222-222222222222"
)

type fakeUsers struct {
	verifyErr, getErr error
	role              string
}

func (f fakeUsers) Verify(_ context.Context, username, password string) (domain.User, error) {
	if f.verifyErr != nil {
		return domain.User{}, f.verifyErr
	}
	if password != "correct-password" {
		return domain.User{}, core.ErrUnauthenticated
	}
	return domain.User{ID: userID, Username: username, Role: core.RoleSuperAdmin}, nil
}
func (f fakeUsers) Get(context.Context, string) (domain.User, error) {
	return domain.User{ID: userID, Role: f.role}, f.getErr
}

type fakeSessions struct {
	sessions map[string]*domain.Session // by hash
	revoked  []string
	rotateOK bool
}

func newSessions() *fakeSessions {
	return &fakeSessions{sessions: map[string]*domain.Session{}, rotateOK: true}
}
func (f *fakeSessions) Create(_ context.Context, user, hash string, expires time.Time) (domain.Session, error) {
	s := &domain.Session{ID: sessionID, UserID: user, ExpiresAt: expires}
	f.sessions[hash] = s
	return *s, nil
}
func (f *fakeSessions) ByRefreshHash(_ context.Context, hash string) (domain.Session, error) {
	s, ok := f.sessions[hash]
	if !ok {
		return domain.Session{}, core.NotFound("session")
	}
	return *s, nil
}
func (f *fakeSessions) Rotate(_ context.Context, _, oldHash, newHash string, expires time.Time) (bool, error) {
	s, ok := f.sessions[oldHash]
	if !ok || !f.rotateOK {
		return false, nil
	}
	delete(f.sessions, oldHash)
	s.ExpiresAt = expires
	f.sessions[newHash] = s
	return true, nil
}
func (f *fakeSessions) Get(_ context.Context, id string) (domain.Session, error) {
	for _, s := range f.sessions {
		if s.ID == id {
			return *s, nil
		}
	}
	return domain.Session{}, core.NotFound("session")
}
func (f *fakeSessions) Revoke(_ context.Context, id string) error {
	f.revoked = append(f.revoked, id)
	for _, s := range f.sessions {
		if s.ID == id {
			s.Revoked = true
		}
	}
	return nil
}

type fakeAttempts struct {
	records map[string]domain.Attempts
	err     error
}

func (f *fakeAttempts) Get(_ context.Context, u string) (domain.Attempts, error) {
	return f.records[u], f.err
}
func (f *fakeAttempts) Save(_ context.Context, u string, a domain.Attempts) error {
	f.records[u] = a
	return nil
}
func (f *fakeAttempts) Reset(_ context.Context, u string) error { delete(f.records, u); return nil }

type fakeSigner struct{ err error }

func (f fakeSigner) Sign(u domain.User, session string, ttl time.Duration) (string, time.Time, error) {
	return "access:" + u.Role + ":" + session, time.Now().Add(ttl), f.err
}

func newService(users fakeUsers) (*Service, *fakeSessions, *fakeAttempts) {
	sessions, attempts := newSessions(), &fakeAttempts{records: map[string]domain.Attempts{}}
	s := New(users, sessions, attempts, fakeSigner{}, Config{})
	n := 0
	s.random = func() (string, error) { n++; return "refresh-" + string(rune('0'+n)), nil }
	return s, sessions, attempts
}

func TestLoginIssuesTokens(t *testing.T) {
	s, sessions, _ := newService(fakeUsers{})
	tokens, err := s.Login(context.Background(), "Admin", "correct-password")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "access:super_admin:"+sessionID || tokens.RefreshToken != "refresh-1" || tokens.SessionID != sessionID {
		t.Fatalf("unexpected tokens %+v", tokens)
	}
	if _, stored := sessions.sessions["refresh-1"]; stored {
		t.Fatal("refresh token must be stored hashed, not in plain text")
	}
	if time.Until(tokens.AccessExpiresAt) > 31*time.Minute {
		t.Fatal("default access ttl should be 30 minutes")
	}
}

func TestLoginLockout(t *testing.T) {
	s, _, attempts := newService(fakeUsers{})
	for i := 0; i < 5; i++ {
		if _, err := s.Login(context.Background(), "admin", "wrong"); !errors.Is(err, core.ErrUnauthenticated) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if _, err := s.Login(context.Background(), "ADMIN", "correct-password"); !errors.Is(err, core.ErrPrecondition) {
		t.Fatalf("locked account signed in: %v", err)
	}
	delete(attempts.records, "admin")
	if _, err := s.Login(context.Background(), "admin", "correct-password"); err != nil {
		t.Fatalf("after unlock: %v", err)
	}
}

func TestLoginErrors(t *testing.T) {
	s, _, _ := newService(fakeUsers{})
	if _, err := s.Login(context.Background(), " ", "x"); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("empty username accepted")
	}
	down, _, _ := newService(fakeUsers{verifyErr: core.Dependency("user service")})
	if _, err := down.Login(context.Background(), "admin", "correct-password"); !errors.Is(err, core.ErrDependency) {
		t.Fatal("user service outage hidden")
	}
	broken, _, attempts := newService(fakeUsers{})
	attempts.err = errors.New("db")
	if _, err := broken.Login(context.Background(), "admin", "correct-password"); err == nil {
		t.Fatal("attempt store error ignored")
	}
	badSigner := New(fakeUsers{}, newSessions(), &fakeAttempts{records: map[string]domain.Attempts{}}, fakeSigner{err: errors.New("sign")}, Config{})
	if _, err := badSigner.Login(context.Background(), "admin", "correct-password"); err == nil {
		t.Fatal("signing error ignored")
	}
	noEntropy, _, _ := newService(fakeUsers{})
	noEntropy.random = func() (string, error) { return "", errors.New("entropy") }
	if _, err := noEntropy.Login(context.Background(), "admin", "correct-password"); err == nil {
		t.Fatal("random failure ignored")
	}
}

func TestRefreshRotatesAndReadsCurrentRole(t *testing.T) {
	s, _, _ := newService(fakeUsers{role: core.RoleViewer})
	first, _ := s.Login(context.Background(), "admin", "correct-password")
	next, err := s.Refresh(context.Background(), first.RefreshToken)
	if err != nil || next.RefreshToken == first.RefreshToken || next.AccessToken != "access:viewer:"+sessionID {
		t.Fatalf("refresh: %+v %v", next, err)
	}
	if _, err := s.Refresh(context.Background(), first.RefreshToken); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("used refresh token accepted twice")
	}
	if _, err := s.Refresh(context.Background(), ""); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("empty refresh token accepted")
	}
}

func TestRefreshRejections(t *testing.T) {
	s, sessions, _ := newService(fakeUsers{role: core.RoleViewer})
	tokens, _ := s.Login(context.Background(), "admin", "correct-password")
	for _, session := range sessions.sessions {
		session.ExpiresAt = time.Now().Add(-time.Minute)
	}
	if _, err := s.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("expired session refreshed")
	}

	deleted, sessions2, _ := newService(fakeUsers{getErr: core.NotFound("user")})
	tokens, _ = deleted.Login(context.Background(), "admin", "correct-password")
	if _, err := deleted.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, core.ErrUnauthenticated) || len(sessions2.revoked) != 1 {
		t.Fatal("deleted user's session must be revoked")
	}

	down, _, _ := newService(fakeUsers{getErr: core.Dependency("down")})
	tokens, _ = down.Login(context.Background(), "admin", "correct-password")
	if _, err := down.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, core.ErrDependency) {
		t.Fatal("user service outage hidden")
	}

	raced, sessions3, _ := newService(fakeUsers{role: core.RoleViewer})
	tokens, _ = raced.Login(context.Background(), "admin", "correct-password")
	sessions3.rotateOK = false
	if _, err := raced.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("lost rotation race must fail")
	}
}

func TestRevoke(t *testing.T) {
	s, sessions, _ := newService(fakeUsers{role: core.RoleViewer})
	tokens, _ := s.Login(context.Background(), "admin", "correct-password")
	if err := s.Revoke(context.Background(), Caller{}, ""); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("anonymous revoke")
	}
	if err := s.Revoke(context.Background(), Caller{ID: "other", Role: core.RoleViewer}, sessionID); !errors.Is(err, core.ErrForbidden) {
		t.Fatal("revoked someone else's session")
	}
	if err := s.Revoke(context.Background(), Caller{ID: userID, SessionID: "bad"}, ""); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad session id accepted")
	}
	if err := s.Revoke(context.Background(), Caller{ID: userID, SessionID: sessionID}, ""); err != nil || len(sessions.revoked) != 1 {
		t.Fatalf("logout: %v", err)
	}
	if _, err := s.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, core.ErrUnauthenticated) {
		t.Fatal("revoked session refreshed")
	}
	if err := s.Revoke(context.Background(), Caller{ID: "a", Role: core.RoleSuperAdmin}, sessionID); err != nil {
		t.Fatalf("admin revoke: %v", err)
	}
}

func TestLockoutPolicy(t *testing.T) {
	p := domain.DefaultLockout
	now := time.Now()
	a := domain.Attempts{}
	for i := 0; i < 4; i++ {
		a = p.AfterFailure(a, now)
	}
	if p.Locked(a, now) || a.Failed != 4 {
		t.Fatal("locked too early")
	}
	// Failures outside the window start a new count.
	if fresh := p.AfterFailure(a, now.Add(16*time.Minute)); fresh.Failed != 1 {
		t.Fatalf("window not reset: %+v", fresh)
	}
	a = p.AfterFailure(a, now)
	if !p.Locked(a, now) || p.Locked(a, now.Add(16*time.Minute)) {
		t.Fatal("lock duration wrong")
	}
}
