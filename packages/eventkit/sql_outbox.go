package eventkit

import (
	"context"
	"database/sql"
	"errors"
)

// SQLOutboxStore reads and acknowledges rows from a service-local transactional outbox.
type SQLOutboxStore struct {
	db *sql.DB
}

func NewSQLOutboxStore(db *sql.DB) (*SQLOutboxStore, error) {
	if db == nil {
		return nil, errors.New("SQL outbox database is required")
	}
	return &SQLOutboxStore{db: db}, nil
}

func (s *SQLOutboxStore) PendingEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id::text, subject, payload::text FROM search_outbox WHERE published_at IS NULL ORDER BY created_at,event_id LIMIT $1`, limit)
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
	_, err := s.db.ExecContext(ctx, `UPDATE search_outbox SET published_at=now() WHERE event_id=$1 AND published_at IS NULL`, eventID)
	return err
}
