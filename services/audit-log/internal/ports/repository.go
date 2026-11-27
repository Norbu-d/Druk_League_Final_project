package ports

import (
	"context"

	"github.com/drukleague/platform/services/audit-log/internal/domain"
)

type Repository interface {
	Record(context.Context, domain.Entry) (domain.Entry, error)
	List(context.Context, string, string, int, int) ([]domain.Entry, bool, error)
}
