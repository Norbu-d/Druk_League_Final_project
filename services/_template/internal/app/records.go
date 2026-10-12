package app

import (
	"context"

	"github.com/drukleague/platform/services/service-template/internal/domain"
	"github.com/drukleague/platform/services/service-template/internal/ports"
)

type RecordService struct{ repo ports.RecordRepository }

func NewRecordService(repo ports.RecordRepository) *RecordService { return &RecordService{repo: repo} }
func (s *RecordService) Create(ctx context.Context, payload []byte) (domain.Record, error) {
	return s.repo.Create(ctx, payload)
}
func (s *RecordService) Get(ctx context.Context, id string) (domain.Record, error) {
	return s.repo.Get(ctx, id)
}