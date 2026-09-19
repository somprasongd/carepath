package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
)

type VisitStep struct {
	Sequence    int    `json:"sequence"`
	ServiceCode string `json:"serviceCode"`
	Status      string `json:"status"`
}

type Visit struct {
	VisitID    string      `json:"visitId"`
	PatientRef string      `json:"patientRef"`
	Status     string      `json:"status"`
	Steps      []VisitStep `json:"steps"`
}

func main() {
	app := fiber.New()

	visits := map[string]Visit{
		"VISIT-001": {
			VisitID:    "VISIT-001",
			PatientRef: "PATIENT-DEMO-001",
			Status:     "ACTIVE",
			Steps: []VisitStep{
				{Sequence: 1, ServiceCode: "REGISTRATION", Status: "COMPLETED"},
				{Sequence: 2, ServiceCode: "SCREENING", Status: "COMPLETED"},
				{Sequence: 3, ServiceCode: "DOCTOR", Status: "COMPLETED"},
				{Sequence: 4, ServiceCode: "LAB", Status: "READY"},
				{Sequence: 5, ServiceCode: "PHARMACY", Status: "PENDING"},
			},
		},
	}

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "mock-his"})
	})

	app.Get("/api/v1/visits/:visitId", func(c fiber.Ctx) error {
		visit, ok := visits[c.Params("visitId")]
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "visit not found"})
		}
		return c.JSON(visit)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	log.Fatal(app.Listen(":" + port))
}
