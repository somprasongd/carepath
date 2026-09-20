package navigation

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the navigation module over HTTP: the route lookup the
// patient navigate screen draws its SVG overlay from (#28).
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the navigation routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Get("/navigation/route", h.route)
}

// route godoc
//
//	@Summary		Get the walking route to a service point
//	@Description	Shortest walkable route for the SVG overlay. from is the current location's navigation node id — the nodeId of a location observation (e.g. I-1301/node-reception); to is the destination service point's code, which the server resolves to the place's entry node before routing.
//	@Tags			navigation
//	@Produce		json
//	@Param			from	query	string	true	"Current location node id (e.g. I-1301/node-reception)"
//	@Param			to		query	string	true	"Destination service point code (e.g. PHARMACY)"
//	@Param			accessibleOnly	query	boolean	false	"Skip non-accessible edges (stairs) so the route detours via the elevator (FR-20)"
//	@Success		200	{object}	navigation.Route
//	@Failure		400	{object}	httpx.ErrorResponse	"missing from or to, or accessibleOnly is not a boolean"
//	@Failure		404	{object}	httpx.ErrorResponse	"unknown node or service point, destination without a mapped place, or no walkable path"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/navigation/route [get]
func (h *Handler) route(c fiber.Ctx) error {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "from and to are required"))
	}
	// AccessibleOnly (#99): absent means the full graph; anything present
	// must parse as a boolean (strconv's forms: true/false/1/0/t/f...), so a
	// typo like "yes" is a 400 rather than a silent full-graph route for a
	// wheelchair user.
	accessibleOnly := false
	if raw := c.Query("accessibleOnly"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return httpx.Error(c, apperr.New(apperr.KindInvalid, "accessibleOnly must be a boolean"))
		}
		accessibleOnly = parsed
	}
	route, err := h.service.RouteToServicePoint(c.Context(), from, to, RouteOptions{AccessibleOnly: accessibleOnly})
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(route)
}
