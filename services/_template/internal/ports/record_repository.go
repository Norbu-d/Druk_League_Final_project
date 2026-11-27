package ports

import (
	"context"

	"github.com/drukleague/platform/services/service-template/internal/domain"
)

type RecordRepository interface {
	Create(context.Context, []byte) (domain.Record, error)
	Get(context.Context, string) (domain.Record, error)
}
