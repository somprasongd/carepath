// Command server runs the Mock HIS, the deterministic fake upstream hospital
// system for development and demo (ADR-0005).
package main

import (
	"log"
	"os"

	"carepath/apps/mock-his/internal/mockhis"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	log.Fatal(mockhis.New().Listen(":" + port))
}
