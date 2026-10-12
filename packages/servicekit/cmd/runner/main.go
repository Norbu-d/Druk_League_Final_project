package main

import (
	"log"
	"os"

	"github.com/drukleague/platform/packages/servicekit/servicekit"
)

func main() {
	name := os.Getenv("SERVICE_NAME")
	if name == "" {
		log.Fatal("SERVICE_NAME is required")
	}
	if err := servicekit.Run(name); err != nil {
		log.Fatal(err)
	}
}
