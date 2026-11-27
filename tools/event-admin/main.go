package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/drukleague/platform/packages/eventkit"
	"github.com/nats-io/nats.go"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	conn, js, err := eventkit.Connect(env("NATS_URL", "nats://localhost:4222"), "drukleague-event-admin")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	switch os.Args[1] {
	case "inspect":
		inspect(js, os.Args[2:])
	case "replay":
		replay(js, os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func inspect(js nats.JetStreamContext, args []string) {
	flags := flag.NewFlagSet("inspect", flag.ExitOnError)
	limit := flags.Uint64("limit", 50, "maximum recent entries to show; 0 shows all retained entries")
	_ = flags.Parse(args)
	info, err := js.StreamInfo(eventkit.DLQStream)
	if err != nil {
		log.Fatalf("read dead-letter stream: %v", err)
	}
	fmt.Printf("stream=%s messages=%d first_seq=%d last_seq=%d retention=30d\n", info.Config.Name, info.State.Msgs, info.State.FirstSeq, info.State.LastSeq)
	if info.State.Msgs == 0 {
		return
	}
	first, last := info.State.FirstSeq, info.State.LastSeq
	if *limit > 0 && last-first+1 > *limit {
		first = last - *limit + 1
	}
	for sequence := first; sequence <= last; sequence++ {
		message, getErr := js.GetMsg(eventkit.DLQStream, sequence)
		if errors.Is(getErr, nats.ErrMsgNotFound) {
			continue
		}
		if getErr != nil {
			log.Fatalf("read dead-letter sequence %d: %v", sequence, getErr)
		}
		fmt.Printf("seq=%d subject=%s original=%s deliveries=%s error=%q bytes=%d\n",
			message.Sequence, message.Subject, message.Header.Get("Drukleague-Original-Subject"),
			message.Header.Get("Drukleague-Deliveries"), message.Header.Get("Drukleague-Error"), len(message.Data))
	}
}

func replay(js nats.JetStreamContext, args []string) {
	flags := flag.NewFlagSet("replay", flag.ExitOnError)
	dryRun := flags.Bool("dry-run", false, "validate and print the replay target without publishing")
	_ = flags.Parse(args)
	if flags.NArg() != 1 {
		log.Fatal("usage: event-admin replay [--dry-run] <dead-letter-sequence>")
	}
	sequence, err := strconv.ParseUint(flags.Arg(0), 10, 64)
	if err != nil || sequence == 0 {
		log.Fatal("dead-letter sequence must be a positive integer")
	}
	message, err := js.GetMsg(eventkit.DLQStream, sequence)
	if err != nil {
		log.Fatalf("read dead-letter sequence %d: %v", sequence, err)
	}
	originalSubject := strings.TrimSpace(message.Header.Get("Drukleague-Original-Subject"))
	if !validSubject(originalSubject) {
		log.Fatalf("dead-letter sequence %d has an invalid original subject", sequence)
	}
	if *dryRun {
		fmt.Printf("would replay seq=%d subject=%s bytes=%d\n", sequence, originalSubject, len(message.Data))
		return
	}
	msgID := fmt.Sprintf("dlq-replay-%d-%d", sequence, time.Now().UnixNano())
	if err := eventkit.Publish(js, originalSubject, message.Data, msgID); err != nil {
		log.Fatalf("publish replay of sequence %d: %v", sequence, err)
	}
	fmt.Printf("replayed seq=%d subject=%s msg_id=%s; original dead-letter retained\n", sequence, originalSubject, msgID)
}

func validSubject(subject string) bool {
	return subject != "" && !strings.ContainsAny(subject, "* >") && !strings.HasPrefix(subject, eventkit.DLQPrefix)
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: event-admin <inspect|replay> [options]")
	fmt.Fprintln(os.Stderr, "Inspect the retained DRUKLEAGUE_DLQ stream or replay one sequence to its original subject.")
}
