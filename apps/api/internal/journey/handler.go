package journey

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the journey module over HTTP.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the journey routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Get("/journeys/:visitId", h.getJourney)
}

// getJourney godoc
//
//	@Summary		Get the patient journey
//	@Description	Returns the CarePath-owned journey projection: steps ordered by sequence, each resolved to its service point, with the deterministic current (first STARTED) and next (first READY) step. A completed visit is reported with completed=true and no current/next.
//	@Tags			journeys
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Success		200	{object}	journey.View
//	@Failure		404	{object}	httpx.ErrorResponse	"no journey projected for this visit"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/{visitId} [get]
func (h *Handler) getJourney(c fiber.Ctx) error {
	view, err := h.service.GetJourney(c.Context(), c.Params("visitId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(view)
}
