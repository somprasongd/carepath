package visit

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the visit module over HTTP.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the visit routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Get("/visits/:visitId", h.getVisit)
	router.Get("/visits/:visitId/next", h.getNextStep)
}

// getVisit godoc
//
//	@Summary		Get normalized visit view
//	@Description	Returns the HIS visit enriched with the next actionable step and its service point.
//	@Tags			visits
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Success		200	{object}	visit.VisitView
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid input"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit not found"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Failure		502	{object}	httpx.ErrorResponse	"upstream HIS error"
//	@Router			/api/v1/visits/{visitId} [get]
func (h *Handler) getVisit(c fiber.Ctx) error {
	view, err := h.service.GetVisitView(c.Context(), c.Params("visitId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(view)
}

// getNextStep godoc
//
//	@Summary		Get next actionable step
//	@Description	Returns the first READY step of the visit resolved to its service point.
//	@Tags			visits
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Success		200	{object}	visit.NextStep
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid input"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit not found or no next actionable step"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Failure		502	{object}	httpx.ErrorResponse	"upstream HIS error"
//	@Router			/api/v1/visits/{visitId}/next [get]
func (h *Handler) getNextStep(c fiber.Ctx) error {
	next, err := h.service.GetNextStep(c.Context(), c.Params("visitId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(next)
}
