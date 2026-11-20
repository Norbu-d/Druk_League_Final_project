package eventkit

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
)

var outboxTablePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// SQLOutboxStore reads and acknowledges rows from a service-local transactional outbox.
type SQLOutboxStore struct {
	db    *sql.DB
	table string
}

func NewSQLOutboxStore(db *sql.DB, table string) (*SQLOutboxStore, error) {
	if db == nil || !outboxTablePattern.MatchString(table) {
		return nil, fmt.Errorf("invalid SQL outbox store configuration")
	}
	return &SQLOutboxStore{db: db, table: table}, nil
}

func (s *SQLOutboxStore) PendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	query := fmt.Sprintf(`SELECT event_id::text, subject, payload::text FROM %s WHERE published_at IS NULL ORDER BY created_at,event_id LIMIT $1`, s.table)
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]OutboxEvent, 0, limit)
	for rows.Next() {
		var event OutboxEvent
		var payload string
		if err := rows.Scan(&event.EventID, &event.Subject, &payload); err != nil {
			return nil, err
		}
		event.Payload = []byte(payload)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *SQLOutboxStore) MarkPublished(ctx context.Context, eventID string) error {
	query := fmt.Sprintf(`UPDATE %s SET published_at=now() WHERE event_id=$1 AND published_at IS NULL`, s.table)
	_, err := s.db.ExecContext(ctx, query, eventID)
	return err
}
