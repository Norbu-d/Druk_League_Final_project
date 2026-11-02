package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func connectEvents(ctx context.Context, db *sql.DB, url string) (*nats.Conn, error) {
	nc, err := nats.Connect(url, nats.Name("drukleague-api"), nats.MaxReconnects(-1), nats.ReconnectWait(time.Second), nats.RetryOnFailedConnect(true))
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	_, err = js.AddStream(&nats.StreamConfig{Name: "DRUKLEAGUE_EVENTS", Subjects: []string{"drukleague.match.finalized"}, Storage: nats.FileStorage, Retention: nats.LimitsPolicy})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		nc.Close()
		return nil, err
	}
	go publishOutbox(ctx, db, js)
	return nc, nil
}

func publishOutbox(ctx context.Context, db *sql.DB, js nats.JetStreamContext) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := db.QueryContext(ctx, `SELECT id,payload::text FROM event_outbox WHERE published_at IS NULL ORDER BY id LIMIT 50`)
			if err != nil {
				log.Printf("event outbox read: %v", err)
				continue
			}
			type entry struct {
				id      int64
				payload string
			}
			batch := []entry{}
			var scanErr error
			for rows.Next() {
				var item entry
				if scanErr = rows.Scan(&item.id, &item.payload); scanErr != nil {
					break
				}
				batch = append(batch, item)
			}
			if scanErr == nil {
				scanErr = rows.Err()
			}
			if closeErr := rows.Close(); closeErr != nil && scanErr == nil {
				scanErr = closeErr
			}
			if scanErr != nil {
				log.Printf("event outbox scan: %v", scanErr)
				continue
			}
			for _, item := range batch {
				_, err = js.Publish("drukleague.match.finalized", []byte(item.payload), nats.MsgId("event-outbox-"+decimal(item.id)))
				if err != nil {
					log.Printf("event publish %d: %v", item.id, err)
					break
				}
				_, err = db.ExecContext(ctx, `UPDATE event_outbox SET published_at=now() WHERE id=$1 AND published_at IS NULL`, item.id)
				if err != nil {
					log.Printf("event mark %d: %v", item.id, err)
					break
				}
			}
		}
	}
}
func decimal(v int64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
