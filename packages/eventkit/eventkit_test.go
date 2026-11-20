package eventkit

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type fakeStreams struct {
	streams   map[string]*nats.StreamConfig
	infoErr   error
	addErr    error
	updateErr error
	added     []string
	updated   []string
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
	if f.addErr != nil {
		return nil, f.addErr
	}
	f.streams[cfg.Name] = cfg
	return &nats.StreamInfo{Config: *cfg}, nil
}
func (f *fakeStreams) UpdateStream(cfg *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	f.updated = append(f.updated, cfg.Name)
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.streams[cfg.Name] = cfg
	return &nats.StreamInfo{Config: *cfg}, nil
}

func TestEnsureStreamsHandlesCreateAndUpdateErrors(t *testing.T) {
	createErr := errors.New("create failed")
	streams := &fakeStreams{streams: map[string]*nats.StreamConfig{}, addErr: createErr}
	if err := EnsureStreams(streams); !errors.Is(err, createErr) {
		t.Fatalf("create error = %v", err)
	}

	streams = &fakeStreams{streams: map[string]*nats.StreamConfig{
		EventsStream: {Name: EventsStream, Subjects: []string{SubjectMatchFinalized}},
		DLQStream:    {Name: DLQStream, Subjects: []string{DLQPrefix + ">"}},
	}, updateErr: errors.New("update failed")}
	if err := EnsureStreams(streams); err == nil || !strings.Contains(err.Error(), "update stream") {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestEnsureStreamsAcceptsConcurrentStreamCreate(t *testing.T) {
	streams := &fakeStreams{streams: map[string]*nats.StreamConfig{}, addErr: nats.ErrStreamNameAlreadyInUse}
	if err := EnsureStreams(streams); err != nil {
		t.Fatalf("concurrent create should be harmless: %v", err)
	}
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
	updatedEvents := 0
	for _, name := range streams.updated {
		if name == EventsStream {
			updatedEvents++
		}
	}
	if updatedEvents != 1 || len(streams.streams[EventsStream].Subjects) != len(EventSubjects) {
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

func TestPublishReturnsPublisherError(t *testing.T) {
	want := errors.New("publish failed")
	if err := Publish(&fakePublisher{err: want}, SubjectFixturesGenerated, []byte(`{}`), "id"); !errors.Is(err, want) {
		t.Fatalf("publish error = %v", err)
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

func TestConnectRejectsMalformedURL(t *testing.T) {
	if _, _, err := Connect("http://[", "eventkit-test"); err == nil {
		t.Fatal("expected malformed server URL to fail")
	}
}

func TestNATSMessageMethods(t *testing.T) {
	message := natsMessage{msg: &nats.Msg{Subject: SubjectSearchTeam, Data: []byte("payload")}}
	if message.Subject() != SubjectSearchTeam || string(message.Data()) != "payload" || message.Deliveries() != 1 {
		t.Fatal("message accessors returned unexpected values")
	}
	// An unbound NATS message has no acknowledgement connection, but each adapter
	// method should still safely return the library error rather than panic.
	_ = message.Ack()
	_ = message.NakWithDelay(time.Second)
	_ = message.Term()
}

func TestDefaults(t *testing.T) {
	c := ConsumerConfig{}
	c.defaults()
	if c.MaxDeliver != 5 || c.Backoff != 2*time.Second {
		t.Fatal("defaults wrong")
	}
}

type fakeOutboxStore struct {
	events       []OutboxEvent
	pendingErr   error
	markErr      error
	markedEvents []string
}

func (s *fakeOutboxStore) PendingEvents(context.Context, int) ([]OutboxEvent, error) {
	return s.events, s.pendingErr
}
func (s *fakeOutboxStore) MarkPublished(_ context.Context, eventID string) error {
	s.markedEvents = append(s.markedEvents, eventID)
	return s.markErr
}

func TestRelayOutboxOncePublishesAndMarksBatch(t *testing.T) {
	store := &fakeOutboxStore{events: []OutboxEvent{{EventID: "one", Subject: SubjectSearchTeam, Payload: []byte(`{"n":1}`)}, {EventID: "two", Subject: SubjectSearchPlayer, Payload: []byte(`{"n":2}`)}}}
	publisher := &fakePublisher{}
	count, err := RelayOutboxOnce(context.Background(), store, publisher, 0)
	if err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if len(publisher.published) != 2 || len(store.markedEvents) != 2 || store.markedEvents[1] != "two" {
		t.Fatalf("published=%d marked=%v", len(publisher.published), store.markedEvents)
	}
	if publisher.published[0].Subject != SubjectSearchTeam || string(publisher.published[0].Data) != `{"n":1}` {
		t.Fatalf("unexpected published event: %+v", publisher.published[0])
	}
}

func TestRelayOutboxOnceReturnsPendingReadError(t *testing.T) {
	want := errors.New("database unavailable")
	count, err := RelayOutboxOnce(context.Background(), &fakeOutboxStore{pendingErr: want}, &fakePublisher{}, 10)
	if count != 0 || !errors.Is(err, want) {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestRelayOutboxOnceStopsWhenPublishFails(t *testing.T) {
	store := &fakeOutboxStore{events: []OutboxEvent{{EventID: "one", Subject: SubjectSearchTeam}, {EventID: "two", Subject: SubjectSearchTeam}}}
	want := errors.New("nats unavailable")
	count, err := RelayOutboxOnce(context.Background(), store, &fakePublisher{err: want}, 2)
	if count != 0 || !errors.Is(err, want) || len(store.markedEvents) != 0 {
		t.Fatalf("count=%d err=%v marked=%v", count, err, store.markedEvents)
	}
}

func TestRelayOutboxOnceReturnsMarkErrorAfterPublish(t *testing.T) {
	want := errors.New("database unavailable")
	store := &fakeOutboxStore{events: []OutboxEvent{{EventID: "one", Subject: SubjectSearchTeam}}, markErr: want}
	publisher := &fakePublisher{}
	count, err := RelayOutboxOnce(context.Background(), store, publisher, 1)
	if count != 0 || !errors.Is(err, want) || len(publisher.published) != 1 {
		t.Fatalf("count=%d err=%v published=%d", count, err, len(publisher.published))
	}
}

func TestRunOutboxRelayStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunOutboxRelay(ctx, &fakeOutboxStore{}, &fakePublisher{}, 0)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("relay did not stop after cancellation")
	}
}

type outboxSQLDriver struct{}
type outboxSQLConn struct{}
type outboxSQLRows struct{ sent bool }
type outboxSQLResult struct{}

var outboxSQLState struct {
	sync.Mutex
	queries []string
	execs   []string
}

func (outboxSQLDriver) Open(string) (driver.Conn, error)  { return outboxSQLConn{}, nil }
func (outboxSQLConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (outboxSQLConn) Close() error                        { return nil }
func (outboxSQLConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }
func (outboxSQLConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	outboxSQLState.Lock()
	outboxSQLState.queries = append(outboxSQLState.queries, query)
	outboxSQLState.Unlock()
	return &outboxSQLRows{}, nil
}
func (outboxSQLConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	outboxSQLState.Lock()
	outboxSQLState.execs = append(outboxSQLState.execs, query)
	outboxSQLState.Unlock()
	return outboxSQLResult{}, nil
}
func (*outboxSQLRows) Columns() []string { return []string{"event_id", "subject", "payload"} }
func (*outboxSQLRows) Close() error      { return nil }
func (r *outboxSQLRows) Next(dest []driver.Value) error {
	if r.sent {
		return io.EOF
	}
	r.sent = true
	dest[0], dest[1], dest[2] = "event-1", SubjectSearchTeam, `{"event_id":"event-1"}`
	return nil
}
func (outboxSQLResult) LastInsertId() (int64, error) { return 0, nil }
func (outboxSQLResult) RowsAffected() (int64, error) { return 1, nil }

var registerOutboxSQLDriver sync.Once

func TestSQLOutboxStoreReadsAndMarksPendingEvents(t *testing.T) {
	registerOutboxSQLDriver.Do(func() { sql.Register("eventkit-outbox-test", outboxSQLDriver{}) })
	db, err := sql.Open("eventkit-outbox-test", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	outboxSQLState.Lock()
	outboxSQLState.queries, outboxSQLState.execs = nil, nil
	outboxSQLState.Unlock()
	store, err := NewSQLOutboxStore(db)
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.PendingEvents(context.Background(), 25)
	if err != nil || len(events) != 1 || events[0].EventID != "event-1" || string(events[0].Payload) != `{"event_id":"event-1"}` {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	if err := store.MarkPublished(context.Background(), events[0].EventID); err != nil {
		t.Fatal(err)
	}
	outboxSQLState.Lock()
	defer outboxSQLState.Unlock()
	if len(outboxSQLState.queries) != 1 || !strings.Contains(outboxSQLState.queries[0], "FROM search_outbox") || len(outboxSQLState.execs) != 1 || !strings.Contains(outboxSQLState.execs[0], "UPDATE search_outbox") {
		t.Fatalf("queries=%v execs=%v", outboxSQLState.queries, outboxSQLState.execs)
	}
}

func TestNewSQLOutboxStoreRejectsNilDB(t *testing.T) {
	if _, err := NewSQLOutboxStore(nil); err == nil {
		t.Fatal("expected nil database error")
	}
}
