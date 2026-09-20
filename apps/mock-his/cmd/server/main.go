// Command server runs the Mock HIS, the deterministic fake upstream hospital
// system for development and demo (ADR-0005).
package main

import (
	"log/slog"
	"os"
	"strings"

	"carepath/apps/mock-his/internal/mockhis"
	"carepath/apps/mock-his/internal/platform/logger"
)

func main() {
	log := logger.New()
	slog.SetDefault(log)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	patientAppBaseURL := strings.TrimRight(os.Getenv("PATIENT_APP_BASE_URL"), "/")
	log.Info("starting mock-his", "addr", ":"+port)
	if err := mockhis.New(log, patientAppBaseURL).Listen(":" + port); err != nil {
		log.Error("server exited", "error", err.Error())
		os.Exit(1)
	}
}
