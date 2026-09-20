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
	// Where the console's pickers read CarePath's service-point list; empty
	// = same-origin (correct behind the prod single-origin proxy).
	carepathAPIBaseURL := strings.TrimRight(os.Getenv("CAREPATH_API_BASE_URL"), "/")
	// The mint wiring (#136): mock-his calls CarePath's link mint the way a
	// real HIS printing a navigation slip does. Server-to-server, so this is
	// the docker-network address of the API, not the browser-facing one.
	carepathInternalBaseURL := strings.TrimRight(os.Getenv("CAREPATH_API_INTERNAL_BASE_URL"), "/")
	if carepathInternalBaseURL == "" {
		carepathInternalBaseURL = "http://api:8080"
	}
	carepathHISAPIKey := os.Getenv("CAREPATH_HIS_API_KEY")
	if carepathHISAPIKey != "" {
		log.Info("visit-link mint configured", "carepath_base", carepathInternalBaseURL)
	}
	log.Info("starting mock-his", "addr", ":"+port)
	if err := mockhis.New(log, patientAppBaseURL, carepathAPIBaseURL,
		carepathInternalBaseURL, carepathHISAPIKey).Listen(":" + port); err != nil {
		log.Error("server exited", "error", err.Error())
		os.Exit(1)
	}
}
