package eventkit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type fakeStreams struct {
	streams map[string]*nats.StreamConfig
	infoErr error
	added   []string
	updated []string
}

func (f *fakeStreams) StreamInfo(name string, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	if f.infoErr != nil {
		return nil, f.infoErr
	}
	config, ok := f.streams[name]
	if !ok {
		return nil, nats.ErrStreamNotFound
	}
	return &nats.StreamInfo{Config: *config}, nil
}
func (f *fakeStreams) AddStream(cfg *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	f.added = append(f.added, cfg.Name)
	f.streams[cfg.Name] = cfg
	return &nats.StreamInfo{Config: *cfg}, nil
}
func (f *fakeStreams) UpdateStream(cfg *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	f.updated = append(f.updated, cfg.Name)
	f.streams[cfg.Name] = cfg
	return &nats.StreamInfo{Config: *cfg}, nil
}

func TestEnsureStreamsCreatesBothStreams(t *testing.T) {
	streams := &fakeStreams{streams: map[string]*nats.StreamConfig{}}
	if err := EnsureStreams(streams); err != nil {
		t.Fatal(err)
	}
	if len(streams.added) != 2 || streams.streams[DLQStream].Subjects[0] != DLQPrefix+">" {
		t.Fatalf("unexpected streams %+v", streams.added)
	}
}

func TestEnsureStreamsAddsMissingSubjectsToLegacyStream(t *testing.T) {
	streams := &fakeStreams{streams: map[string]*nats.StreamConfig{
		EventsStream: {Name: EventsStream, Subjects: []string{SubjectMatchFinalized}},
		DLQStream:    {Name: DLQStream, Subjects: []string{DLQPrefix + ">"}},
	}}
	if err := EnsureStreams(streams); err != nil {
		t.Fatal(err)
	}
	if len(streams.updated) != 1 || len(streams.streams[EventsStream].Subjects) != 2 {
		t.Fatalf("legacy stream not updated: %+v", streams.streams[EventsStream].Subjects)
	}
	// Running again changes nothing.
	streams.updated = nil
	if err := EnsureStreams(streams); err != nil || len(streams.updated) != 0 {
		t.Fatalf("second run should be a no-op: %v %v", err, streams.updated)
	}
}

func TestEnsureStreamsReportsReadErrors(t *testing.T) {
	streams := &fakeStreams{streams: map[string]*nats.StreamConfig{}, infoErr: errors.New("nats down")}
	if err := EnsureStreams(streams); err == nil {
		t.Fatal("expected error")
	}
}

type fakePublisher struct {
	published []*nats.Msg
	err       error
}

func (f *fakePublisher) PublishMsg(m *nats.Msg, _ ...nats.PubOpt) (*nats.PubAck, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.published = append(f.published, m)
	return &nats.PubAck{}, nil
}

func TestPublish(t *testing.T) {
	publisher := &fakePublisher{}
	if err := Publish(publisher, SubjectFixturesGenerated, []byte(`{}`), "id-1"); err != nil {
		t.Fatal(err)
	}
	if err := Publish(publisher, SubjectFixturesGenerated, []byte(`{}`), ""); err != nil {
		t.Fatal(err)
	}
	if len(publisher.published) != 2 || publisher.published[0].Subject != SubjectFixturesGenerated {
		t.Fatal("message not published")
	}
}

type fakeMessage struct {
	deliveries uint64
	acked      bool
	naked      time.Duration
	termed     bool
}

func (m *fakeMessage) Subject() string                    { return SubjectMatchFinalized }
func (m *fakeMessage) Data() []byte                       { return []byte(`{"event_id":"e1"}`) }
func (m *fakeMessage) Deliveries() uint64                 { return m.deliveries }
func (m *fakeMessage) Ack() error                         { m.acked = true; return nil }
func (m *fakeMessage) NakWithDelay(d time.Duration) error { m.naked = d; return nil }
func (m *fakeMessage) Term() error                        { m.termed = true; return nil }

func config(err error) ConsumerConfig {
	return ConsumerConfig{Durable: "test-v1", Subject: SubjectMatchFinalized, MaxDeliver: 3, Backoff: time.Second,
		Handler: func(context.Context, string, []byte) error { return err }}
}

func TestProcessAcksSuccess(t *testing.T) {
	message := &fakeMessage{deliveries: 1}
	if outcome := Process(context.Background(), message, &fakePublisher{}, config(nil)); outcome != Acked || !message.acked {
		t.Fatalf("outcome %s", outcome)
	}
}

func TestProcessRetriesTransientErrorsWithBackoff(t *testing.T) {
	message := &fakeMessage{deliveries: 2}
	if outcome := Process(context.Background(), message, &fakePublisher{}, config(errors.New("db down"))); outcome != Retried || message.naked != 2*time.Second {
		t.Fatalf("outcome %s delay %s", outcome, message.naked)
	}
}

func TestProcessDeadLettersAfterMaxDeliveries(t *testing.T) {
	message := &fakeMessage{deliveries: 3}
	dlq := &fakePublisher{}
	if outcome := Process(context.Background(), message, dlq, config(errors.New("still failing"))); outcome != DeadLettered || !message.termed {
		t.Fatalf("outcome %s", outcome)
	}
	dead := dlq.published[0]
	if dead.Subject != DLQPrefix+"test-v1" || dead.Header.Get("Drukleague-Deliveries") != "3" || dead.Header.Get("Drukleague-Original-Subject") != SubjectMatchFinalized || dead.Header.Get("Drukleague-Error") != "still failing" {
		t.Fatalf("unexpected dead letter %+v", dead)
	}
}

func TestProcessDeadLettersPoisonMessagesImmediately(t *testing.T) {
	message := &fakeMessage{deliveries: 1}
	dlq := &fakePublisher{}
	if outcome := Process(context.Background(), message, dlq, config(Permanent(errors.New("bad json")))); outcome != DeadLettered || len(dlq.published) != 1 {
		t.Fatalf("outcome %s", outcome)
	}
}

func TestProcessKeepsMessageWhenDeadLetterPublishFails(t *testing.T) {
	message := &fakeMessage{deliveries: 3}
	if outcome := Process(context.Background(), message, &fakePublisher{err: errors.New("nats down")}, config(errors.New("x"))); outcome != Retried || message.termed {
		t.Fatalf("outcome %s termed %v", outcome, message.termed)
	}
}

func TestPermanentWrapping(t *testing.T) {
	if Permanent(nil) != nil {
		t.Fatal("nil must stay nil")
	}
	base := errors.New("bad")
	wrapped := Permanent(base)
	if !IsPermanent(wrapped) || !errors.Is(wrapped, base) || wrapped.Error() != "bad" || IsPermanent(base) {
		t.Fatal("permanent wrapping wrong")
	}
}

func TestStartConsumerValidatesConfig(t *testing.T) {
	if err := StartConsumer(context.Background(), nil, ConsumerConfig{}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDefaults(t *testing.T) {
	c := ConsumerConfig{}
	c.defaults()
	if c.MaxDeliver != 5 || c.Backoff != 2*time.Second {
		t.Fatal("defaults wrong")
	}
}
