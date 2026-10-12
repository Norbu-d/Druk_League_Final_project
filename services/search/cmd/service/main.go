package main

import (
	"log"

	"github.com/drukleague/platform/packages/servicekit/servicekit"
)

func main() {
	if err := servicekit.Run("search-service"); err != nil {
		log.Fatal(err)
	}
}
