// Package eventkit holds the shared NATS JetStream setup: stream definitions, a durable pull
// consumer with bounded redelivery and a dead-letter stream, and a de-duplicated publisher.
package eventkit

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	EventsStream = "DRUKLEAGUE_EVENTS"
	DLQStream    = "DRUKLEAGUE_DLQ"

	SubjectMatchFinalized    = "drukleague.match.finalized"
	SubjectFixturesGenerated = "drukleague.fixtures.generated"
	// DLQPrefix + durable consumer name is where a consumer parks messages it cannot process.
	DLQPrefix = "drukleague.dlq."
)

// EventSubjects are the subjects carried by the events stream.
var EventSubjects = []string{SubjectMatchFinalized, SubjectFixturesGenerated}

// Connect opens a connection that keeps retrying, so a service can start before NATS is ready.
func Connect(url, name string) (*nats.Conn, nats.JetStreamContext, error) {
	connection, err := nats.Connect(url, nats.Name(name), nats.MaxReconnects(-1), nats.ReconnectWait(time.Second), nats.RetryOnFailedConnect(true),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Printf("nats disconnected (%s): %v", name, err)
			}
		}),
		nats.ReconnectHandler(func(*nats.Conn) { log.Printf("nats reconnected (%s)", name) }))
	if err != nil {
		return nil, nil, err
	}
	js, err := connection.JetStream()
	if err != nil {
		connection.Close()
		return nil, nil, err
	}
	return connection, js, nil
}

// StreamManager is the subset of nats.JetStreamContext used to declare streams.
type StreamManager interface {
	StreamInfo(stream string, opts ...nats.JSOpt) (*nats.StreamInfo, error)
	AddStream(cfg *nats.StreamConfig, opts ...nats.JSOpt) (*nats.StreamInfo, error)
	UpdateStream(cfg *nats.StreamConfig, opts ...nats.JSOpt) (*nats.StreamInfo, error)
}

// EnsureStreams creates the events and dead-letter streams, adding any missing subjects to an
// existing events stream (older services created it with only the MatchFinalized subject).
func EnsureStreams(js StreamManager) error {
	if err := ensureStream(js, &nats.StreamConfig{Name: EventsStream, Subjects: EventSubjects, Storage: nats.FileStorage, Retention: nats.LimitsPolicy, Duplicates: 2 * time.Minute}); err != nil {
		return err
	}
	// Dead letters are kept for 30 days for inspection and manual replay.
	return ensureStream(js, &nats.StreamConfig{Name: DLQStream, Subjects: []string{DLQPrefix + ">"}, Storage: nats.FileStorage, Retention: nats.LimitsPolicy, MaxAge: 30 * 24 * time.Hour})
}

func ensureStream(js StreamManager, want *nats.StreamConfig) error {
	info, err := js.StreamInfo(want.Name)
	if errors.Is(err, nats.ErrStreamNotFound) {
		if _, err = js.AddStream(want); err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
			return fmt.Errorf("create stream %s: %w", want.Name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read stream %s: %w", want.Name, err)
	}
	missing := missingSubjects(info.Config.Subjects, want.Subjects)
	if len(missing) == 0 {
		return nil
	}
	config := info.Config
	config.Subjects = append(append([]string{}, config.Subjects...), missing...)
	if _, err = js.UpdateStream(&config); err != nil {
		return fmt.Errorf("update stream %s subjects: %w", want.Name, err)
	}
	return nil
}

func missingSubjects(have, want []string) []string {
	present := make(map[string]bool, len(have))
	for _, subject := range have {
		present[subject] = true
	}
	missing := []string{}
	for _, subject := range want {
		if !present[subject] {
			missing = append(missing, subject)
		}
	}
	return missing
}

// Publisher is the subset of nats.JetStreamContext used to publish.
type Publisher interface {
	PublishMsg(m *nats.Msg, opts ...nats.PubOpt) (*nats.PubAck, error)
}

// Publish sends one event; msgID lets JetStream drop duplicates inside the dedupe window.
func Publish(js Publisher, subject string, payload []byte, msgID string) error {
	options := []nats.PubOpt{}
	if msgID != "" {
		options = append(options, nats.MsgId(msgID))
	}
	_, err := js.PublishMsg(&nats.Msg{Subject: subject, Data: payload}, options...)
	return err
}

// permanentError marks a message that will never succeed (malformed or semantically invalid).
type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent wraps err so the consumer dead-letters the message at once instead of retrying.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

// IsPermanent reports whether err was wrapped with Permanent.
func IsPermanent(err error) bool {
	var permanent permanentError
	return errors.As(err, &permanent)
}

// Handler processes one message payload. Returning nil acknowledges it.
type Handler func(ctx context.Context, subject string, payload []byte) error

// ConsumerConfig describes one durable pull consumer.
type ConsumerConfig struct {
	Durable    string
	Subject    string
	MaxDeliver int           // default 5
	Backoff    time.Duration // first redelivery delay, doubled per attempt; default 2s
	Handler    Handler
}

func (c *ConsumerConfig) defaults() {
	if c.MaxDeliver <= 0 {
		c.MaxDeliver = 5
	}
	if c.Backoff <= 0 {
		c.Backoff = 2 * time.Second
	}
}

// StartConsumer binds a durable pull consumer and processes messages until ctx is cancelled.
func StartConsumer(ctx context.Context, js nats.JetStreamContext, config ConsumerConfig) error {
	config.defaults()
	if config.Durable == "" || config.Subject == "" || config.Handler == nil {
		return errors.New("consumer durable name, subject and handler are required")
	}
	sub, err := js.PullSubscribe(config.Subject, config.Durable, nats.BindStream(EventsStream), nats.AckExplicit(), nats.MaxDeliver(config.MaxDeliver), nats.AckWait(30*time.Second))
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", config.Durable, err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			messages, err := sub.Fetch(10, nats.MaxWait(time.Second))
			if err != nil {
				if !errors.Is(err, nats.ErrTimeout) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, nats.ErrNoResponders) {
					log.Printf("%s fetch: %v", config.Durable, err)
					time.Sleep(time.Second)
				}
				continue
			}
			for _, message := range messages {
				Process(ctx, natsMessage{message}, js, config)
			}
		}
	}()
	return nil
}

// Message is the part of a JetStream message the consumer needs; it lets tests use fakes.
type Message interface {
	Subject() string
	Data() []byte
	Deliveries() uint64
	Ack() error
	NakWithDelay(time.Duration) error
	Term() error
}

type natsMessage struct{ msg *nats.Msg }

func (m natsMessage) Subject() string { return m.msg.Subject }
func (m natsMessage) Data() []byte    { return m.msg.Data }
func (m natsMessage) Deliveries() uint64 {
	meta, err := m.msg.Metadata()
	if err != nil {
		return 1
	}
	return meta.NumDelivered
}
func (m natsMessage) Ack() error                         { return m.msg.Ack() }
func (m natsMessage) NakWithDelay(d time.Duration) error { return m.msg.NakWithDelay(d) }
func (m natsMessage) Term() error                        { return m.msg.Term() }

// Outcome is what the consumer did with a message.
type Outcome string

const (
	Acked        Outcome = "acked"
	Retried      Outcome = "retried"
	DeadLettered Outcome = "dead-lettered"
)

// Process runs the handler for one message and acks, schedules a retry, or dead-letters it.
// A message is dead-lettered when the handler returns a Permanent error or when it has been
// delivered MaxDeliver times; it is then terminated so JetStream stops redelivering it.
func Process(ctx context.Context, message Message, dlq Publisher, config ConsumerConfig) Outcome {
	config.defaults()
	err := config.Handler(ctx, message.Subject(), message.Data())
	if err == nil {
		_ = message.Ack()
		return Acked
	}
	deliveries := message.Deliveries()
	if !IsPermanent(err) && deliveries < uint64(config.MaxDeliver) {
		log.Printf("%s retry %d/%d: %v", config.Durable, deliveries, config.MaxDeliver, err)
		_ = message.NakWithDelay(config.Backoff * time.Duration(1<<min(deliveries-1, 6)))
		return Retried
	}
	dead := &nats.Msg{Subject: DLQPrefix + config.Durable, Data: message.Data(), Header: nats.Header{}}
	dead.Header.Set("Drukleague-Original-Subject", message.Subject())
	dead.Header.Set("Drukleague-Error", err.Error())
	dead.Header.Set("Drukleague-Deliveries", strconv.FormatUint(deliveries, 10))
	if _, publishErr := dlq.PublishMsg(dead); publishErr != nil {
		// Keep the message in the events stream rather than lose it; it will be redelivered.
		log.Printf("%s dead-letter publish failed: %v", config.Durable, publishErr)
		_ = message.NakWithDelay(config.Backoff)
		return Retried
	}
	log.Printf("%s dead-lettered after %d deliveries: %v", config.Durable, deliveries, err)
	_ = message.Term()
	return DeadLettered
}
