package connector

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drukleague/platform/packages/eventkit"
)

func TestInvokesFunctionAndPublishesResult(t *testing.T) {
	var gotBody, gotSubject string
	fn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody, gotSubject = string(b), r.Header.Get("X-Event-Subject")
		_, _ = w.Write([]byte(`{"headline":"Paro 2–1 Haa"}`))
	}))
	defer fn.Close()
	var published, publishedOn string
	inv := Invoker{Function: "match-report", URL: fn.URL, Client: fn.Client(), Publish: func(subject string, data []byte) error {
		published, publishedOn = string(data), subject
		return nil
	}}
	if err := inv.Handle(context.Background(), eventkit.SubjectMatchFinalized, []byte(`{"match_id":"m"}`)); err != nil {
		t.Fatal(err)
	}
	if gotBody != `{"match_id":"m"}` || gotSubject != eventkit.SubjectMatchFinalized {
		t.Fatalf("function received %q on %q", gotBody, gotSubject)
	}
	if publishedOn != "drukleague.functions.match-report" || published != `{"headline":"Paro 2–1 Haa"}` {
		t.Fatalf("published %q on %q", published, publishedOn)
	}
}

func TestRetryVersusDeadLetter(t *testing.T) {
	status := http.StatusInternalServerError
	fn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer fn.Close()
	inv := Invoker{Function: "match-report", URL: fn.URL, Client: fn.Client()}
	for _, s := range []int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusTooManyRequests} {
		status = s
		if err := inv.Handle(context.Background(), "s", nil); err == nil || eventkit.IsPermanent(err) {
			t.Fatalf("%d should be retried, got %v", s, err)
		}
	}
	status = http.StatusBadRequest
	if err := inv.Handle(context.Background(), "s", nil); !eventkit.IsPermanent(err) {
		t.Fatalf("400 should go to the dead-letter stream, got %v", err)
	}
	fn.Close()
	if err := inv.Handle(context.Background(), "s", nil); err == nil || eventkit.IsPermanent(err) {
		t.Fatalf("an unreachable function should be retried, got %v", err)
	}
	bad := Invoker{Function: "x", URL: "://bad", Client: http.DefaultClient}
	if err := bad.Handle(context.Background(), "s", nil); !eventkit.IsPermanent(err) {
		t.Fatalf("a malformed URL is a permanent error, got %v", err)
	}
}

func TestPublishFailureDoesNotReplayTheEvent(t *testing.T) {
	fn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer fn.Close()
	inv := Invoker{Function: "match-report", URL: fn.URL, Client: fn.Client(), Publish: func(string, []byte) error { return errors.New("nats down") }}
	if err := inv.Handle(context.Background(), "s", nil); err != nil {
		t.Fatalf("the function ran, so the event should be acked: %v", err)
	}
}

func TestEnvFallback(t *testing.T) {
	t.Setenv("FUNCTION_NAME", "")
	if Env("FUNCTION_NAME", "match-report") != "match-report" {
		t.Fatal("empty value should use the fallback")
	}
	t.Setenv("FUNCTION_NAME", "standings")
	if Env("FUNCTION_NAME", "match-report") != "standings" {
		t.Fatal("set value should win")
	}
}
