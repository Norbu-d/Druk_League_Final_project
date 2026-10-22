package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/user/internal/domain"
)

const userID = "11111111-1111-4111-8111-111111111111"

type fakeRepo struct {
	users     map[string]string // username -> hash
	updated   string
	updateErr error
	hasAdmin  bool
}

func (f *fakeRepo) Create(_ context.Context, u domain.User, hash string) (domain.User, error) {
	if _, ok := f.users[u.Username]; ok {
		return domain.User{}, core.Conflict("username already exists")
	}
	f.users[u.Username] = hash
	u.ID = userID
	return u, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (domain.User, error) {
	if id != userID {
		return domain.User{}, core.NotFound("user not found")
	}
	return domain.User{ID: id, Role: core.RoleMatchOfficial}, nil
}
func (f *fakeRepo) Credentials(_ context.Context, username string) (domain.User, string, error) {
	hash, ok := f.users[username]
	if !ok {
		return domain.User{}, "", core.NotFound("user not found")
	}
	return domain.User{ID: userID, Username: username, Role: core.RoleSuperAdmin}, hash, nil
}
func (f *fakeRepo) UpdatePasswordHash(_ context.Context, _, hash string) error {
	f.updated = hash
	return f.updateErr
}
func (f *fakeRepo) List(context.Context, string, int, int) ([]domain.User, bool, error) {
	return []domain.User{{ID: userID}}, false, nil
}
func (f *fakeRepo) CreateFirstSuperAdmin(_ context.Context, username, hash string) (bool, error) {
	if f.hasAdmin {
		return false, nil
	}
	f.users[username], f.hasAdmin = hash, true
	return true, nil
}

// fakeHasher stores "hash:<password>"; "legacy:<password>" asks for an upgrade.
type fakeHasher struct {
	verified []string
	err      error
}

func (f *fakeHasher) Hash(password string) (string, error) { return "hash:" + password, f.err }
func (f *fakeHasher) Verify(hash, password string) (bool, bool) {
	f.verified = append(f.verified, hash)
	switch hash {
	case "hash:" + password:
		return true, false
	case "legacy:" + password:
		return true, true
	}
	return false, false
}

var (
	admin   = Caller{ID: "a", Role: core.RoleSuperAdmin}
	service = Caller{ID: "service:auth-service", Role: core.RoleService}
)

func newService() (*Service, *fakeRepo, *fakeHasher) {
	repo, hasher := &fakeRepo{users: map[string]string{}}, &fakeHasher{}
	return New(repo, hasher), repo, hasher
}

func TestCreate(t *testing.T) {
	s, repo, _ := newService()
	u, err := s.Create(context.Background(), admin, " karma.sonam ", "long-enough-password", core.RoleMatchOfficial)
	if err != nil || u.Username != "karma.sonam" || repo.users["karma.sonam"] != "hash:long-enough-password" {
		t.Fatalf("create: %+v %v", u, err)
	}
	if _, err := s.Create(context.Background(), service, "team.boss", "long-enough-password", core.RoleTeamManager); err != nil {
		t.Fatalf("self-service team manager: %v", err)
	}
	cases := []struct {
		name     string
		caller   Caller
		username string
		password string
		role     string
		want     error
	}{
		{"viewer creates", Caller{Role: core.RoleViewer}, "someone", "long-enough-password", core.RoleViewer, core.ErrForbidden},
		{"admin creates admin", admin, "boss", "long-enough-password", core.RoleSuperAdmin, core.ErrInvalid},
		{"service creates official", service, "ref", "long-enough-password", core.RoleMatchOfficial, core.ErrInvalid},
		{"short password", admin, "someone", "short", core.RoleViewer, core.ErrInvalid},
		{"long password", admin, "someone", strings.Repeat("x", 73), core.RoleViewer, core.ErrInvalid},
		{"bad username", admin, "a b", "long-enough-password", core.RoleViewer, core.ErrInvalid},
		{"duplicate", admin, "karma.sonam", "long-enough-password", core.RoleViewer, core.ErrConflict},
	}
	for _, c := range cases {
		if _, err := s.Create(context.Background(), c.caller, c.username, c.password, c.role); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
	}
	failing, _, hasher := newService()
	hasher.err = errors.New("entropy")
	if _, err := failing.Create(context.Background(), admin, "someone", "long-enough-password", core.RoleViewer); err == nil {
		t.Fatal("hash failure ignored")
	}
}

func TestGetAndList(t *testing.T) {
	s, _, _ := newService()
	if _, err := s.Get(context.Background(), admin, "x"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if _, err := s.Get(context.Background(), Caller{ID: "other", Role: core.RoleViewer}, userID); !errors.Is(err, core.ErrForbidden) {
		t.Fatal("viewer read another account")
	}
	for _, c := range []Caller{admin, service, {ID: userID, Role: core.RoleMatchOfficial}} {
		if _, err := s.Get(context.Background(), c, userID); err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
	}
	if _, _, err := s.List(context.Background(), Caller{Role: core.RoleMatchOfficial}, "", 10, 0); !errors.Is(err, core.ErrForbidden) {
		t.Fatal("official listed accounts")
	}
	if _, _, err := s.List(context.Background(), admin, "dzongkhag_admin", 10, 0); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("unknown role filter accepted")
	}
	if users, _, err := s.List(context.Background(), admin, core.RoleViewer, 10, 0); err != nil || len(users) != 1 {
		t.Fatalf("list: %v", err)
	}
}

func TestVerifyCredentials(t *testing.T) {
	s, repo, hasher := newService()
	repo.users["admin"] = "hash:drukleague-dev"
	if u, err := s.VerifyCredentials(context.Background(), service, " admin ", "drukleague-dev"); err != nil || u.Role != core.RoleSuperAdmin {
		t.Fatalf("verify: %+v %v", u, err)
	}
	if _, err := s.VerifyCredentials(context.Background(), admin, "admin", "drukleague-dev"); !errors.Is(err, core.ErrForbidden) {
		t.Fatal("non-service caller verified credentials")
	}
	for _, c := range [][2]string{{"admin", "wrong"}, {"", "x"}, {"admin", ""}, {"admin", strings.Repeat("x", 73)}} {
		if _, err := s.VerifyCredentials(context.Background(), service, c[0], c[1]); !errors.Is(err, core.ErrUnauthenticated) {
			t.Fatalf("%v accepted", c)
		}
	}
	hasher.verified = nil
	if _, err := s.VerifyCredentials(context.Background(), service, "ghost", "whatever-password"); !errors.Is(err, core.ErrUnauthenticated) || len(hasher.verified) != 1 || hasher.verified[0] != "" {
		t.Fatalf("unknown user must still hash once: %v %v", err, hasher.verified)
	}
	repo.users["legacy"] = "legacy:old-password-1"
	if _, err := s.VerifyCredentials(context.Background(), service, "legacy", "old-password-1"); err != nil || repo.updated != "hash:old-password-1" {
		t.Fatalf("legacy hash not upgraded: %v %q", err, repo.updated)
	}
	repo.updateErr = errors.New("db")
	if _, err := s.VerifyCredentials(context.Background(), service, "legacy", "old-password-1"); err != nil {
		t.Fatal("upgrade failure must not fail login")
	}
}

func TestBootstrap(t *testing.T) {
	s, repo, _ := newService()
	if created, err := s.Bootstrap(context.Background(), "admin", "drukleague-dev"); err != nil || !created || repo.users["admin"] == "" {
		t.Fatalf("bootstrap: %v %v", created, err)
	}
	if created, err := s.Bootstrap(context.Background(), "admin", "drukleague-dev"); err != nil || created {
		t.Fatalf("second bootstrap must do nothing: %v %v", created, err)
	}
	if _, err := s.Bootstrap(context.Background(), "x", "drukleague-dev"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad username accepted")
	}
	if _, err := s.Bootstrap(context.Background(), "admin", "short"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("short password accepted")
	}
}
