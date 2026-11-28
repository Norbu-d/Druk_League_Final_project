package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/drukleague/platform/functions/connector/internal/connector"
	"github.com/drukleague/platform/packages/eventkit"
)

func main() {
	nc, js, err := eventkit.Connect(connector.Env("NATS_URL", "nats://nats:4222"), "drukleague-function-connector")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()
	if err := eventkit.EnsureStreams(js); err != nil {
		log.Fatal(err)
	}
	invoker := connector.Invoker{
		Function: connector.Env("FUNCTION_NAME", "match-report"),
		URL:      connector.Env("FUNCTION_URL", "http://fn-match-report:8080"),
		Client:   &http.Client{Timeout: 15 * time.Second},
		Publish:  nc.Publish,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = eventkit.StartConsumer(ctx, js, eventkit.ConsumerConfig{
		Durable: "function-" + invoker.Function,
		Subject: eventkit.SubjectMatchFinalized,
		Handler: invoker.Handle,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("function connector: %s -> %s (%s)", eventkit.SubjectMatchFinalized, invoker.URL, invoker.Function)
	<-ctx.Done()
}
