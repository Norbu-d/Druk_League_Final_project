package main

import (
	"log"

	"github.com/drukleague/platform/packages/servicekit/servicekit"
)

func main() {
	if err := servicekit.Run("configuration-service"); err != nil {
		log.Fatal(err)
	}
}
