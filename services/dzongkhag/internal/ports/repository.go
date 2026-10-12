package ports

import (
	"context"

	"github.com/drukleague/platform/services/dzongkhag/internal/domain"
)

type Repository interface {
	Get(context.Context, string) (domain.Dzongkhag, error)
	List(context.Context) ([]domain.Dzongkhag, error)
}
