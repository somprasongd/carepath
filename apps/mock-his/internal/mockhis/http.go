package mockhis

import (
	_ "embed"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

//go:embed console.html
var consoleHTML []byte

// New builds the Mock HIS HTTP app implementing the contract in
// packages/contracts/openapi/mock-his.yaml, plus the demo-driver surface
// (/api/v1/demo/* and /console) that exists only on the mock — a real HIS
// has its own operator tooling (see docs/integration/mock-his.md).
func New() *fiber.App {
	store := NewStore()
	app := fiber.New()

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "mock-his"})
	})

	// Console: single embedded page, same origin as the API.
	app.Get("/", func(c fiber.Ctx) error {
		return c.Redirect().Status(http.StatusFound).To("/console")
	})
	app.Get("/console", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.Send(consoleHTML)
	})

	app.Get("/api/v1/visits/:visitId", func(c fiber.Ctx) error {
		visit, ok := store.GetVisit(c.Params("visitId"))
		if !ok {
			return errResponse(c, http.StatusNotFound, "visit not found")
		}
		return c.JSON(visit)
	})

	app.Post("/api/v1/visits/:visitId/steps/:sequence/transition", func(c fiber.Ctx) error {
		sequence, err := strconv.Atoi(c.Params("sequence"))
		if err != nil || sequence < 1 {
			return errResponse(c, http.StatusBadRequest, "invalid step sequence")
		}
		var cmd TransitionCommand
		if err := c.Bind().Body(&cmd); err != nil {
			return errResponse(c, http.StatusBadRequest, "invalid request body")
		}
		if cmd.CommandID == "" {
			return errResponse(c, http.StatusBadRequest, "commandId is required")
		}
		step, terr := store.Transition(c.Params("visitId"), sequence, cmd)
		if terr != nil {
			switch terr.Kind {
			case ErrNotFound:
				return errResponse(c, http.StatusNotFound, terr.Msg)
			case ErrConflict:
				return errResponse(c, http.StatusConflict, terr.Msg)
			default:
				return errResponse(c, http.StatusBadRequest, terr.Msg)
			}
		}
		return c.JSON(step)
	})

	app.Get("/api/v1/events", func(c fiber.Ctx) error {
		limit := 50
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				return errResponse(c, http.StatusBadRequest, "invalid limit")
			}
			limit = parsed
		}
		events, next := store.Events(c.Query("after"), limit)
		return c.JSON(fiber.Map{"events": events, "nextAfter": next})
	})

	// Demo-driver API (mock-only, deliberately outside the canonical
	// contract): what the console uses to stage a demo. State changes here
	// and via the transition command above are announced as canonical
	// events, so CarePath still learns everything through the feed.
	app.Get("/api/v1/demo/visits", func(c fiber.Ctx) error {
		return c.JSON(store.ListVisits())
	})

	app.Post("/api/v1/demo/visits", func(c fiber.Ctx) error {
		var body struct {
			PatientRef   string   `json:"patientRef"`
			ServiceCodes []string `json:"serviceCodes"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return errResponse(c, http.StatusBadRequest, "invalid request body")
		}
		visit, verr := store.CreateVisit(body.PatientRef, body.ServiceCodes)
		if verr != nil {
			return errResponse(c, http.StatusBadRequest, verr.Msg)
		}
		return c.Status(http.StatusCreated).JSON(visit)
	})

	app.Post("/api/v1/demo/visits/:visitId/orders", func(c fiber.Ctx) error {
		var body struct {
			ServiceCode string `json:"serviceCode"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return errResponse(c, http.StatusBadRequest, "invalid request body")
		}
		step, verr := store.AddOrder(c.Params("visitId"), body.ServiceCode)
		if verr != nil {
			switch verr.Kind {
			case ErrNotFound:
				return errResponse(c, http.StatusNotFound, verr.Msg)
			case ErrConflict:
				return errResponse(c, http.StatusConflict, verr.Msg)
			default:
				return errResponse(c, http.StatusBadRequest, verr.Msg)
			}
		}
		return c.Status(http.StatusCreated).JSON(step)
	})

	return app
}

func errResponse(c fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}
