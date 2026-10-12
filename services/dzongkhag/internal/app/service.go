package app

import (
	"context"

	"github.com/drukleague/platform/services/dzongkhag/internal/domain"
	"github.com/drukleague/platform/services/dzongkhag/internal/ports"
)

type Service struct{ repo ports.Repository }

func New(repo ports.Repository) *Service { return &Service{repo: repo} }
func (s *Service) Get(ctx context.Context, code string) (domain.Dzongkhag, error) {
	return s.repo.Get(ctx, code)
}
func (s *Service) List(ctx context.Context) ([]domain.Dzongkhag, error) {
	return s.repo.List(ctx)
}
