package eventkit

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type OutboxEvent struct {
	EventID string
	Subject string
	Payload []byte
}

type OutboxStore interface {
	PendingEvents(context.Context, int) ([]OutboxEvent, error)
	MarkPublished(context.Context, string) error
}

// StartSQLOutboxRelay connects to NATS, ensures the shared streams exist, and
// starts publishing the SQL outbox until ctx is cancelled. The returned closer
// owns the NATS connection.
func StartSQLOutboxRelay(ctx context.Context, db *sql.DB, natsURL, clientName string) (interface{ Close() }, error) {
	conn, js, err := Connect(natsURL, clientName)
	if err != nil {
		return nil, err
	}
	return startSQLOutboxRelay(ctx, db, conn, js, js)
}

func startSQLOutboxRelay(ctx context.Context, db *sql.DB, conn interface{ Close() }, streams StreamManager, publisher Publisher) (interface{ Close() }, error) {
	if err := EnsureStreams(streams); err != nil {
		conn.Close()
		return nil, err
	}
	store, err := NewSQLOutboxStore(db)
	if err != nil {
		conn.Close()
		return nil, err
	}
	go RunOutboxRelay(ctx, store, publisher, time.Second)
	return conn, nil
}

// RelayOutboxOnce publishes one bounded batch. Delivery is at least once; consumers must dedupe by event ID.
func RelayOutboxOnce(ctx context.Context, store OutboxStore, publisher Publisher, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 50
	}
	events, err := store.PendingEvents(ctx, batchSize)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, event := range events {
		if err := Publish(publisher, event.Subject, event.Payload, ""); err != nil {
			return published, err
		}
		if err := store.MarkPublished(ctx, event.EventID); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func RunOutboxRelay(ctx context.Context, store OutboxStore, publisher Publisher, interval time.Duration) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := RelayOutboxOnce(ctx, store, publisher, 50); err != nil {
			log.Printf("search-index outbox relay: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
