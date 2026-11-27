package ports

import (
	"context"

	"github.com/drukleague/platform/services/notification/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.Notification) (domain.Notification, bool, error)
	List(context.Context, string, int, int) ([]domain.Notification, bool, error)
}

type TeamGetter interface {
	Get(context.Context, string) (domain.Team, error)
}
