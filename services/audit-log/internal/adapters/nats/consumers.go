package natsadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/services/audit-log/internal/domain"
)

type Recorder interface {
	RecordAdminAction(context.Context, domain.AdminAction) error
	RecordMatchFinalized(context.Context, domain.MatchFinalized) error
}

func MatchFinalizedConsumer(service Recorder) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{Durable: "audit-log-match-finalized-v1", Subject: eventkit.SubjectMatchFinalized,
		MaxDeliver: 5, Backoff: 2 * time.Second, Handler: handler(service, eventkit.SubjectMatchFinalized)}
}

func AdminActionsConsumer(service Recorder) eventkit.ConsumerConfig {
	return eventkit.ConsumerConfig{Durable: "audit-log-admin-actions-v1", Subject: eventkit.SubjectAdminActions,
		MaxDeliver: 5, Backoff: 2 * time.Second, Handler: handler(service, eventkit.SubjectAdminActions)}
}

func handler(service Recorder, filter string) eventkit.Handler {
	return func(ctx context.Context, subject string, payload []byte) error {
		switch filter {
		case eventkit.SubjectMatchFinalized:
			if subject != eventkit.SubjectMatchFinalized {
				return eventkit.Permanent(fmt.Errorf("unexpected subject %q", subject))
			}
			var event domain.MatchFinalized
			if err := json.Unmarshal(payload, &event); err != nil {
				return eventkit.Permanent(fmt.Errorf("decode MatchFinalized event: %w", err))
			}
			if err := event.Validate(); err != nil {
				return eventkit.Permanent(err)
			}
			return service.RecordMatchFinalized(ctx, event)
		default:
			if !subjectMatchesAdmin(subject) {
				return eventkit.Permanent(fmt.Errorf("unexpected admin event subject %q", subject))
			}
			var event domain.AdminAction
			if err := json.Unmarshal(payload, &event); err != nil {
				return eventkit.Permanent(fmt.Errorf("decode admin action event: %w", err))
			}
			if err := event.Validate(); err != nil {
				return eventkit.Permanent(err)
			}
			return service.RecordAdminAction(ctx, event)
		}
	}
}

func subjectMatchesAdmin(subject string) bool {
	return len(subject) > len("drukleague.admin.") && subject[:len("drukleague.admin.")] == "drukleague.admin."
}
