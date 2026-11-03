package ports

import (
	"context"

	"github.com/drukleague/platform/services/search/internal/domain"
)

type Repository interface {
	Search(context.Context, string, []string, int, int) ([]domain.Hit, error)
}
