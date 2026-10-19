package app

import (
	"context"
	"strings"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/venue/internal/domain"
	"github.com/drukleague/platform/services/venue/internal/ports"
)

type Service struct {
	repo       ports.Repository
	dzongkhags ports.Dzongkhags
}

func New(repo ports.Repository, dzongkhags ports.Dzongkhags) *Service {
	return &Service{repo: repo, dzongkhags: dzongkhags}
}

// Create registers a venue; only the tournament admin manages venues.
func (s *Service) Create(ctx context.Context, role string, v domain.Venue) (domain.Venue, error) {
	if role != core.RoleSuperAdmin {
		return domain.Venue{}, core.Forbidden("only the tournament admin can add venues")
	}
	v, err := v.Normalize()
	if err != nil {
		return domain.Venue{}, err
	}
	if v.Dzongkhag, err = s.dzongkhags.Canonical(ctx, v.Dzongkhag); err != nil {
		return domain.Venue{}, err
	}
	return s.repo.Create(ctx, v)
}

func (s *Service) Get(ctx context.Context, id string) (domain.Venue, error) {
	if !core.IsUUID(id) {
		return domain.Venue{}, core.Invalid("venue id must be a UUID")
	}
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) List(ctx context.Context, dzongkhag string, limit, offset int) ([]domain.Venue, bool, error) {
	dzongkhag = strings.TrimSpace(dzongkhag)
	if dzongkhag != "" {
		canonical, err := s.dzongkhags.Canonical(ctx, dzongkhag)
		if err != nil {
			return nil, false, err
		}
		dzongkhag = canonical
	}
	return s.repo.List(ctx, dzongkhag, limit, offset)
}
