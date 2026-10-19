package app

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/user/internal/domain"
	"github.com/drukleague/platform/services/user/internal/ports"
)

type Caller struct{ ID, Role string }

type Service struct {
	repo   ports.Repository
	hasher ports.Hasher
}

func New(repo ports.Repository, hasher ports.Hasher) *Service {
	return &Service{repo: repo, hasher: hasher}
}

func (s *Service) Create(ctx context.Context, caller Caller, username, password, role string) (domain.User, error) {
	role = strings.TrimSpace(role)
	if !domain.CreatableRole(caller.Role, role) {
		if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleService {
			return domain.User{}, core.Forbidden("only the tournament admin can create accounts")
		}
		return domain.User{}, core.Invalid("role must be match_official, team_manager or viewer")
	}
	username, err := domain.NormalizeUsername(username)
	if err != nil {
		return domain.User{}, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return domain.User{}, err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}
	return s.repo.Create(ctx, domain.User{Username: username, Role: role}, hash)
}

// Get returns a user to themselves, the admin, or another service (lookups by Match and Auth).
func (s *Service) Get(ctx context.Context, caller Caller, id string) (domain.User, error) {
	id = strings.TrimSpace(id)
	if !core.IsUUID(id) {
		return domain.User{}, core.Invalid("user id must be a UUID")
	}
	if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleService && caller.ID != id {
		return domain.User{}, core.Forbidden("cannot read another user account")
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, caller Caller, role string, limit, offset int) ([]domain.User, bool, error) {
	if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleService {
		return nil, false, core.Forbidden("only the tournament admin can list accounts")
	}
	role = strings.TrimSpace(role)
	if role != "" && !core.IsUserRole(role) {
		return nil, false, core.Invalid("unsupported role filter")
	}
	return s.repo.List(ctx, role, limit, offset)
}

// VerifyCredentials checks a username and password for the Auth service. A matching legacy
// PBKDF2 hash is upgraded to bcrypt; failure to upgrade does not fail the login.
func (s *Service) VerifyCredentials(ctx context.Context, caller Caller, username, password string) (domain.User, error) {
	if caller.Role != core.RoleService {
		return domain.User{}, core.Forbidden("only internal services may verify credentials")
	}
	invalid := core.ErrUnauthenticated
	username = strings.TrimSpace(username)
	if username == "" || password == "" || len(password) > domain.MaxPasswordBytes {
		return domain.User{}, invalid
	}
	user, hash, err := s.repo.Credentials(ctx, username)
	if errors.Is(err, core.ErrNotFound) {
		// Hash anyway so a missing username takes as long as a wrong password.
		_, _ = s.hasher.Verify("", password)
		return domain.User{}, invalid
	}
	if err != nil {
		return domain.User{}, err
	}
	ok, rehash := s.hasher.Verify(hash, password)
	if !ok {
		return domain.User{}, invalid
	}
	if rehash {
		if upgraded, err := s.hasher.Hash(password); err == nil {
			if err := s.repo.UpdatePasswordHash(ctx, user.ID, upgraded); err != nil {
				log.Printf("password hash upgrade for %s failed: %v", user.ID, err)
			}
		}
	}
	return user, nil
}

// Bootstrap creates the first super admin from configuration if none exists.
func (s *Service) Bootstrap(ctx context.Context, username, password string) (bool, error) {
	username, err := domain.NormalizeUsername(username)
	if err != nil {
		return false, err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return false, err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return false, err
	}
	return s.repo.CreateFirstSuperAdmin(ctx, username, hash)
}
