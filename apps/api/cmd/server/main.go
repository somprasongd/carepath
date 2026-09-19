package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
)

type HISVisitStep struct {
	Sequence    int    `json:"sequence"`
	ServiceCode string `json:"serviceCode"`
	Status      string `json:"status"`
}

type HISVisit struct {
	VisitID    string         `json:"visitId"`
	PatientRef string         `json:"patientRef"`
	Status     string         `json:"status"`
	Steps      []HISVisitStep `json:"steps"`
}

type ServicePoint struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	PlaceID string `json:"placeId"`
}

type VisitView struct {
	HISVisit
	Next *struct {
		Sequence     int           `json:"sequence"`
		Status       string        `json:"status"`
		ServicePoint *ServicePoint `json:"servicePoint,omitempty"`
	} `json:"next,omitempty"`
}

var servicePoints = map[string]ServicePoint{
	"REGISTRATION": {ID: "SP-REG", Code: "REGISTRATION", Name: "Registration", PlaceID: "REG-01"},
	"LAB":          {ID: "SP-LAB", Code: "LAB", Name: "Laboratory", PlaceID: "LAB-01"},
	"PHARMACY":     {ID: "SP-PHARMACY", Code: "PHARMACY", Name: "Pharmacy", PlaceID: "PHARMACY-01"},
}

func main() {
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})
	client := &http.Client{Timeout: 5 * time.Second}

	hisBaseURL := os.Getenv("HIS_BASE_URL")
	if hisBaseURL == "" {
		hisBaseURL = "http://localhost:8090"
	}

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "carepath-api"})
	})

	app.Get("/api/v1/visits/:visitId", func(c fiber.Ctx) error {
		visit, status, err := getVisit(client, hisBaseURL, c.Params("visitId"))
		if err != nil {
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(buildVisitView(visit))
	})

	app.Get("/api/v1/visits/:visitId/next", func(c fiber.Ctx) error {
		visit, status, err := getVisit(client, hisBaseURL, c.Params("visitId"))
		if err != nil {
			return c.Status(status).JSON(fiber.Map{"error": err.Error()})
		}
		view := buildVisitView(visit)
		if view.Next == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no next actionable step"})
		}
		return c.JSON(view.Next)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(app.Listen(":" + port))
}

func getVisit(client *http.Client, baseURL, visitID string) (HISVisit, int, error) {
	var visit HISVisit
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/visits/%s", baseURL, visitID))
	if err != nil {
		return visit, fiber.StatusBadGateway, fmt.Errorf("HIS unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return visit, fiber.StatusNotFound, fmt.Errorf("visit not found")
	}
	if resp.StatusCode != http.StatusOK {
		return visit, fiber.StatusBadGateway, fmt.Errorf("HIS returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&visit); err != nil {
		return visit, fiber.StatusBadGateway, fmt.Errorf("invalid HIS response: %w", err)
	}
	return visit, fiber.StatusOK, nil
}

func buildVisitView(visit HISVisit) VisitView {
	view := VisitView{HISVisit: visit}
	for _, step := range visit.Steps {
		if step.Status != "READY" {
			continue
		}
		sp, ok := servicePoints[step.ServiceCode]
		view.Next = &struct {
			Sequence     int           `json:"sequence"`
			Status       string        `json:"status"`
			ServicePoint *ServicePoint `json:"servicePoint,omitempty"`
		}{Sequence: step.Sequence, Status: step.Status}
		if ok {
			view.Next.ServicePoint = &sp
		}
		break
	}
	return view
}
