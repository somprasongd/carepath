package servicepoint

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/auth"
	"carepath/apps/api/internal/platform/apperr"
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
// The public reads stay unguarded; myServicePoints is staff-only and takes
// the staff guard injected by the composition root (ADR-0010 — per-route,
// never group-wide).
func (h *Handler) Register(router fiber.Router, staffGuard fiber.Handler) {
	router.Get("/service-points", h.list)
	router.Get("/service-points/:code", h.getByCode)
	router.Get("/staff/my/service-points", staffGuard, h.myServicePoints)
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

// myServicePoints godoc
//
//	@Summary		List the service points the caller may work
//	@Description	FR-15/#102: the caller's assigned service points (user_service_point), with resolved place and floor — the picker for the staff queue console. ADMIN sees every active point; STAFF sees only the points assigned to them.
//	@Tags			staff
//	@Security		bearerAuth
//	@Produce		json
//	@Success		200	{array}	servicepoint.ServicePoint
//	@Failure		401	{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403	{object}	httpx.ErrorResponse	"authenticated, but the user's roles do not allow this action"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/staff/my/service-points [get]
func (h *Handler) myServicePoints(c fiber.Ctx) error {
	principal := auth.PrincipalFromContext(c.Context())
	if principal.UserID == "" {
		// The staff guard resolves the principal before the handler runs;
		// arriving without one means the guard chain was mounted wrong.
		return httpx.Error(c, apperr.New(apperr.KindUnauthorized, "missing principal"))
	}
	var (
		points []ServicePoint
		err    error
	)
	if principal.HasRole(auth.RoleAdmin) {
		points, err = h.service.List(c.Context())
	} else {
		points, err = h.service.ListForUser(c.Context(), principal.UserID)
	}
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(points)
}
