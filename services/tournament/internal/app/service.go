package app

import (
	"context"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/tournament/internal/domain"
	"github.com/drukleague/platform/services/tournament/internal/ports"
)

type Service struct {
	repo       ports.Repository
	dzongkhags ports.Dzongkhags
}

func New(repo ports.Repository, dzongkhags ports.Dzongkhags) *Service {
	return &Service{repo: repo, dzongkhags: dzongkhags}
}

// Create opens a tournament for registration; only the tournament admin creates tournaments.
func (s *Service) Create(ctx context.Context, role string, t domain.Tournament) (domain.Tournament, error) {
	if role != core.RoleSuperAdmin {
		return domain.Tournament{}, core.Forbidden("only the tournament admin can create tournaments")
	}
	t, err := t.Normalize()
	if err != nil {
		return domain.Tournament{}, err
	}
	if t.Dzongkhag, err = s.dzongkhags.Canonical(ctx, t.Dzongkhag); err != nil {
		return domain.Tournament{}, err
	}
	t.Status = domain.StatusRegistrationOpen
	return s.repo.Create(ctx, t)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Tournament, error) {
	if !core.IsUUID(id) {
		return domain.Tournament{}, core.Invalid("tournament id must be a UUID")
	}
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context, season string, limit, offset int) ([]domain.Tournament, bool, error) {
	return s.repo.List(ctx, strings.TrimSpace(season), limit, offset)
}
