package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

const adminActionSubjectPrefix = "drukleague.admin."

type auditEventTx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// recordAdminAction adds the audit event to the same transaction as its admin change.
func recordAdminAction(ctx context.Context, tx auditEventTx, actorID, action, resourceType, resourceID, reason string) error {
	var eventID string
	if err := tx.QueryRowContext(ctx, `SELECT gen_random_uuid()::text`).Scan(&eventID); err != nil {
		return err
	}
	payload, err := json.Marshal(adminActionEvent{
		EventID: eventID, ActorID: actorID, Action: action, ResourceType: resourceType,
		ResourceID: resourceID, Reason: reason, OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO event_outbox(subject,payload) VALUES($1,$2::jsonb)`, adminActionSubjectPrefix+action, string(payload))
	return err
}
