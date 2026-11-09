package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/drukleague/platform/internal/auditoutbox"
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
	if err = ensureEventStream(js); err != nil {
		nc.Close()
		return nil, err
	}
	go publishOutbox(ctx, db, js)
	return nc, nil
}

func ensureEventStream(js nats.JetStreamContext) error {
	wanted := []string{"drukleague.match.finalized", "drukleague.fixtures.generated", "drukleague.admin.>"}
	info, err := js.StreamInfo("DRUKLEAGUE_EVENTS")
	if errors.Is(err, nats.ErrStreamNotFound) {
		_, err = js.AddStream(&nats.StreamConfig{
			Name: "DRUKLEAGUE_EVENTS", Subjects: wanted,
			Storage: nats.FileStorage, Retention: nats.LimitsPolicy,
		})
		if errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
			return ensureEventStream(js)
		}
		return err
	}
	if err != nil {
		return err
	}

	config := info.Config
	for _, subject := range wanted {
		found := false
		for _, existing := range config.Subjects {
			if existing == subject {
				found = true
				break
			}
		}
		if !found {
			config.Subjects = append(config.Subjects, subject)
		}
	}
	_, err = js.UpdateStream(&config)
	return err
}

func publishOutbox(ctx context.Context, db *sql.DB, js nats.JetStreamContext) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var pruneAt time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Now().After(pruneAt) {
				if _, err := db.ExecContext(ctx, `DELETE FROM event_outbox WHERE published_at < now() - interval '30 days'`); err != nil {
					log.Printf("event outbox retention cleanup: %v", err)
				} else {
					pruneAt = time.Now().Add(time.Hour)
				}
			}
			rows, err := db.QueryContext(ctx, `SELECT id,subject,payload::text FROM event_outbox WHERE published_at IS NULL ORDER BY id LIMIT 50`)
			if err != nil {
				log.Printf("event outbox read: %v", err)
				continue
			}
			type entry struct {
				id      int64
				subject string
				payload string
			}
			batch := []entry{}
			var scanErr error
			for rows.Next() {
				var item entry
				if scanErr = rows.Scan(&item.id, &item.subject, &item.payload); scanErr != nil {
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
				subject, subjectErr := eventSubject(item.subject)
				if subjectErr != nil {
					log.Printf("event outbox subject %d: %v", item.id, subjectErr)
					continue
				}
				_, err = js.Publish(subject, []byte(item.payload), nats.MsgId("event-outbox-"+decimal(item.id)))
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

func eventSubject(subject string) (string, error) {
	return auditoutbox.Subject(subject)
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
