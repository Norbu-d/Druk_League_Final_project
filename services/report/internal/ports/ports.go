package ports

import (
	"context"

	"github.com/drukleague/platform/services/report/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.Report) (domain.Report, error)
	Get(context.Context, string) (domain.Report, error)
}

type Standings interface {
	Get(context.Context, string) ([]domain.Standing, error)
}

type Fixtures interface {
	List(context.Context, string, int32, string) ([]domain.Fixture, string, error)
}
