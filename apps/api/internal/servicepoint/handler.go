package servicepoint

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the service point module over HTTP.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the service point routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Get("/service-points", h.list)
	router.Get("/service-points/:code", h.getByCode)
}

// list godoc
//
//	@Summary		List service points
//	@Description	Active service points with their resolved place and floor — the service → destination mapping patients navigate by and staff see in the console.
//	@Tags			service-points
//	@Produce		json
//	@Success		200	{array}	servicepoint.ServicePoint
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/service-points [get]
func (h *Handler) list(c fiber.Ctx) error {
	points, err := h.service.List(c.Context())
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(points)
}

// getByCode godoc
//
//	@Summary		Get a service point by code
//	@Description	One active service point (e.g. code LAB) with its resolved place and floor — the destination lookup for a care step's service code.
//	@Tags			service-points
//	@Produce		json
//	@Param			code	path	string	true	"Service code (e.g. REGISTRATION, DOCTOR, LAB, XRAY, PHARMACY)"
//	@Success		200	{object}	servicepoint.ServicePoint
//	@Failure		404	{object}	httpx.ErrorResponse	"service point not found"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/service-points/{code} [get]
func (h *Handler) getByCode(c fiber.Ctx) error {
	sp, err := h.service.GetByCode(c.Context(), c.Params("code"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(sp)
}
