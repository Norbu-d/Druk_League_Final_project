package app

import (
	"context"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/audit-log/internal/domain"
	"github.com/drukleague/platform/services/audit-log/internal/ports"
)

type Caller struct{ ID, Role string }

type Service struct{ repo ports.Repository }

func New(repo ports.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Record(ctx context.Context, caller Caller, entry domain.Entry) (domain.Entry, error) {
	if caller.Role != core.RoleSuperAdmin && caller.Role != core.RoleService {
		return domain.Entry{}, core.Forbidden("only admins and internal services can record audit entries")
	}
	entry.ActorID = strings.TrimSpace(caller.ID)
	entry.Action = strings.TrimSpace(entry.Action)
	entry.ResourceType = strings.TrimSpace(entry.ResourceType)
	entry.ResourceID = strings.TrimSpace(entry.ResourceID)
	entry.Reason = strings.TrimSpace(entry.Reason)
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}
	if err := entry.Validate(); err != nil {
		return domain.Entry{}, err
	}
	return s.repo.Record(ctx, entry)
}

func (s *Service) List(ctx context.Context, caller Caller, resourceType, resourceID string, limit, offset int) ([]domain.Entry, bool, error) {
	if caller.Role != core.RoleSuperAdmin {
		return nil, false, core.Forbidden("only administrators can list audit entries")
	}
	resourceType = strings.TrimSpace(resourceType)
	resourceID = strings.TrimSpace(resourceID)
	if (resourceType == "") != (resourceID == "") {
		return nil, false, core.Invalid("resource type and resource id must be supplied together")
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, false, core.Invalid("page size must be between 1 and 100 and offset cannot be negative")
	}
	return s.repo.List(ctx, resourceType, resourceID, limit, offset)
}

func (s *Service) RecordAdminAction(ctx context.Context, event domain.AdminAction) error {
	if err := event.Validate(); err != nil {
		return err
	}
	return s.recordEvent(ctx, domain.Entry{EventID: strings.TrimSpace(event.EventID), ActorID: strings.TrimSpace(event.ActorID),
		Action: strings.TrimSpace(event.Action), ResourceType: strings.TrimSpace(event.ResourceType),
		ResourceID: strings.TrimSpace(event.ResourceID), Reason: strings.TrimSpace(event.Reason), OccurredAt: event.OccurredAt.UTC()})
}

func (s *Service) RecordMatchFinalized(ctx context.Context, event domain.MatchFinalized) error {
	if err := event.Validate(); err != nil {
		return err
	}
	actor := strings.TrimSpace(event.FinalizedBy)
	if actor == "" {
		actor = "system"
	}
	return s.recordEvent(ctx, domain.Entry{EventID: strings.TrimSpace(event.EventID), ActorID: actor,
		Action: "match.finalized", ResourceType: "match", ResourceID: strings.TrimSpace(event.MatchID),
		Reason: strings.TrimSpace(event.Reason), OccurredAt: event.FinalizedAt.UTC()})
}

func (s *Service) recordEvent(ctx context.Context, entry domain.Entry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	_, err := s.repo.Record(ctx, entry)
	return err
}
