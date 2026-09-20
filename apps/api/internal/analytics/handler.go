package analytics

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the analytics module over HTTP: the executive overview
// (#86). The composition root decides the auth policy — STAFF, ADMIN, and
// EXECUTIVE all reach it; an executive token reaches nothing else.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the analytics routes under the given /api/v1 router,
// behind the given per-route guard.
func (h *Handler) Register(router fiber.Router, guard fiber.Handler) {
	router.Get("/analytics/overview", guard, h.overview)
}

// overview godoc
//
//	@Summary		Get the executive analytics overview
//	@Description	Operational metrics for the executive dashboard (#86): per-service-point wait/service numbers and visit-wide summary, aggregated in SQL from the append-only step timeline (#85). window=today means midnight-to-now in the analytics timezone (Asia/Bangkok by default). Executives get EXECUTIVE logins that reach this surface and nothing else (ADR-0010); no patient-level data (NFR-03).
//	@Tags			analytics
//	@Produce		json
//	@Param			window	query	string	false	"Aggregation window; only \"today\" exists in the MVP"	Enums(today)
//	@Success		200	{object}	analytics.Overview
//	@Failure		400	{object}	httpx.ErrorResponse	"unsupported window"
//	@Failure		401	{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403	{object}	httpx.ErrorResponse	"authenticated, but the user's roles do not allow this action"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/analytics/overview [get]
func (h *Handler) overview(c fiber.Ctx) error {
	window := c.Query("window", WindowToday)
	overview, err := h.service.Overview(c.Context(), window)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(overview)
}
