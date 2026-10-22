package eventkit

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// startJetStream runs an embedded NATS server with JetStream for the test.
func startJetStream(t *testing.T) string {
	t.Helper()
	srv, err := server.NewServer(&server.Options{Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir(), NoLog: true, NoSigs: true})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("embedded NATS did not start")
	}
	t.Cleanup(srv.Shutdown)
	return srv.ClientURL()
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestConsumerAcksAndDeadLettersAgainstJetStream(t *testing.T) {
	url := startJetStream(t)
	connection, js, err := Connect(url, "eventkit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	// A legacy stream with one subject is upgraded in place.
	if _, err := js.AddStream(&nats.StreamConfig{Name: EventsStream, Subjects: []string{SubjectMatchFinalized}}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureStreams(js); err != nil {
		t.Fatal(err)
	}
	info, _ := js.StreamInfo(EventsStream)
	if len(info.Config.Subjects) != len(EventSubjects) {
		t.Fatalf("stream subjects not upgraded: %v", info.Config.Subjects)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var handled atomic.Int32
	err = StartConsumer(ctx, js, ConsumerConfig{Durable: "eventkit-test-v1", Subject: SubjectMatchFinalized, MaxDeliver: 2, Backoff: 10 * time.Millisecond,
		Handler: func(_ context.Context, _ string, payload []byte) error {
			handled.Add(1)
			if string(payload) == "poison" {
				return errors.New("cannot process")
			}
			return nil
		}})
	if err != nil {
		t.Fatal(err)
	}
	if err := Publish(js, SubjectMatchFinalized, []byte("good"), "m-1"); err != nil {
		t.Fatal(err)
	}
	// The same message id inside the duplicate window is dropped by JetStream.
	if err := Publish(js, SubjectMatchFinalized, []byte("good"), "m-1"); err != nil {
		t.Fatal(err)
	}
	if err := Publish(js, SubjectMatchFinalized, []byte("poison"), "m-2"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "dead letter", func() bool {
		dlq, err := js.StreamInfo(DLQStream)
		return err == nil && dlq.State.Msgs == 1
	})
	// good once, poison twice (MaxDeliver=2).
	waitFor(t, "handler calls", func() bool { return handled.Load() == 3 })
	dead, err := js.GetLastMsg(DLQStream, DLQPrefix+"eventkit-test-v1")
	if err != nil {
		t.Fatal(err)
	}
	if string(dead.Data) != "poison" || dead.Header.Get("Drukleague-Deliveries") != "2" {
		t.Fatalf("unexpected dead letter %q %v", dead.Data, dead.Header)
	}
}
