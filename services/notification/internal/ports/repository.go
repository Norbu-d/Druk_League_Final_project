package ports

import (
	"context"

	"github.com/drukleague/platform/services/notification/internal/domain"
)

type Team struct{ ID, Name, Dzongkhag, ManagerID string }

type TeamDirectory interface {
	GetTeam(context.Context, string) (Team, error)
}

type Repository interface {
	Create(context.Context, domain.Notification) (domain.Notification, error)
	List(context.Context, string, int, int) ([]domain.Notification, bool, error)
}
