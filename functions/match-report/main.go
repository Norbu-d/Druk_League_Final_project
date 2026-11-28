package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

// The OpenFaaS of-watchdog runs this process and forwards each invocation to it
// (mode=http, upstream_url=http://127.0.0.1:8082).
func main() {
	api := os.Getenv("DRUKLEAGUE_API")
	if api == "" {
		api = "http://bff:8080/api"
	}
	handler := Handler{API: api, Client: &http.Client{Timeout: 5 * time.Second}}
	server := &http.Server{Addr: "127.0.0.1:8082", Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServe())
}
