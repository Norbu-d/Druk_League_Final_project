// Package connector is the DrukLeague function connector: the OpenFaaS "connector" pattern for
// NATS JetStream. It reads events from a durable JetStream consumer and invokes a function
// over HTTP, so the function itself stays stateless and knows nothing about NATS.
package connector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/drukleague/platform/packages/eventkit"
)

// Invoker forwards one event to one function.
type Invoker struct {
	Function string // function name, used in the result subject and logs
	URL      string // e.g. http://fn-match-report:8080 or http://gateway:8080/function/match-report
	Client   *http.Client
	// Publish, when set, receives the function's response on drukleague.functions.<name>.
	Publish func(subject string, data []byte) error
}

// ResultSubject is where a function's output is published for other consumers.
func (i Invoker) ResultSubject() string { return "drukleague.functions." + i.Function }

// Handle implements eventkit.Handler. Network failures, 5xx and 429 responses return an
// error so eventkit redelivers with backoff; other 4xx responses mean the event itself is
// unusable and are marked permanent, so eventkit moves the message to the dead-letter stream.
func (i Invoker) Handle(ctx context.Context, subject string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.URL, bytes.NewReader(payload))
	if err != nil {
		return eventkit.Permanent(fmt.Errorf("%s: build request: %w", i.Function, err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-Subject", subject)
	resp, err := i.Client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: invoke: %w", i.Function, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	switch {
	case resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%s: function returned %s", i.Function, resp.Status)
	case resp.StatusCode >= 400:
		return eventkit.Permanent(fmt.Errorf("%s: event rejected (%s): %s", i.Function, resp.Status, bytes.TrimSpace(body)))
	}
	log.Printf("%s: %s", i.Function, bytes.TrimSpace(body))
	if i.Publish != nil {
		if err := i.Publish(i.ResultSubject(), body); err != nil {
			// The function ran; a lost notification of its output must not replay the event.
			log.Printf("%s: publish result: %v", i.Function, err)
		}
	}
	return nil
}

// Env returns the environment variable, or fallback when it is unset or empty.
func Env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
