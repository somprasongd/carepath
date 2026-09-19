package mockhis

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// New builds the Mock HIS HTTP app implementing the contract in
// packages/contracts/openapi/mock-his.yaml.
func New() *fiber.App {
	store := NewStore()
	app := fiber.New()

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "mock-his"})
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

	return app
}

func errResponse(c fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}
