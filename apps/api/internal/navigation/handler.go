package navigation

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the navigation module over HTTP: the route lookup the
// patient navigate screen draws its SVG overlay from (#28), and the graph
// itself, which apps/web used to import from packages/floorplans at build
// time (#105).
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the navigation routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Get("/navigation/route", h.route)
	router.Get("/navigation/amenities", h.amenities)
	router.Get("/navigation/nodes", h.nodes)
	router.Get("/navigation/edges", h.edges)
}

// nodes godoc
//
//	@Summary		List navigation nodes
//	@Description	#105/ADR-0015: the graph's nodes, which apps/web used to read from a build-time copy of packages/floorplans/graphs. The QR stickers staff print for lifts, stairs and entrances are derived from this list, so a stale copy sends patients to nodes that no longer exist — which is why the copy had to go.
//	@Tags			navigation
//	@Produce		json
//	@Success		200	{array}		navigation.NavNode
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/navigation/nodes [get]
func (h *Handler) nodes(c fiber.Ctx) error {
	nodes, err := h.service.ListNodes(c.Context())
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(nodes)
}

// edges godoc
//
//	@Summary		List navigation edges
//	@Description	#105/ADR-0015: the graph's directed, costed edges. Two-way connections are two rows (A→B and B→A) and a cross-floor transition is an edge whose endpoints sit on different floors, so this is the whole walkable graph, not a per-floor slice.
//	@Tags			navigation
//	@Produce		json
//	@Success		200	{array}		navigation.NavEdge
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/navigation/edges [get]
func (h *Handler) edges(c fiber.Ctx) error {
	edges, err := h.service.ListEdges(c.Context())
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(edges)
}

// route godoc
//
//	@Summary		Get the walking route to a service point or place
//	@Description	Shortest walkable route for the SVG overlay. from is the current location's navigation node id — the nodeId of a location observation (e.g. I-1301/node-reception). The destination is exactly one of to (a service point's code, resolved to the place's entry node before routing) or toPlace (a place id, resolved the same way — how amenity destinations from /navigation/amenities are routed, #109).
//	@Tags			navigation
//	@Produce		json
//	@Param			from	query	string	true	"Current location node id (e.g. I-1301/node-reception)"
//	@Param			to		query	string	false	"Destination service point code (e.g. PHARMACY) — exactly one of to or toPlace"
//	@Param			toPlace	query	string	false	"Destination place id (e.g. RESTROOM-01) — exactly one of to or toPlace"
//	@Param			accessibleOnly	query	boolean	false	"Skip non-accessible edges (stairs) so the route detours via the elevator (FR-20)"
//	@Success		200	{object}	navigation.Route
//	@Failure		400	{object}	httpx.ErrorResponse	"missing from, missing or duplicated destination, or accessibleOnly is not a boolean"
//	@Failure		404	{object}	httpx.ErrorResponse	"unknown node, service point or place, destination without a mapped place, or no walkable path"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/navigation/route [get]
func (h *Handler) route(c fiber.Ctx) error {
	from := c.Query("from")
	to := c.Query("to")
	toPlace := c.Query("toPlace")
	// Exactly one destination: both given is ambiguous, neither is missing
	// the whole question — and `to` alone stays the canonical form the
	// journey's navigate screen has always used (#109 adds toPlace for
	// amenity destinations, which have no service point code).
	if from == "" || (to == "") == (toPlace == "") {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "from and exactly one of to or toPlace are required"))
	}
	accessibleOnly, err := parseAccessibleOnly(c)
	if err != nil {
		return httpx.Error(c, err)
	}
	var route Route
	if to != "" {
		route, err = h.service.RouteToServicePoint(c.Context(), from, to, RouteOptions{AccessibleOnly: accessibleOnly})
	} else {
		route, err = h.service.RouteToPlace(c.Context(), from, toPlace, RouteOptions{AccessibleOnly: accessibleOnly})
	}
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(route)
}

// amenities godoc
//
//	@Summary		Search amenities near a location
//	@Description	#109/FR-26: amenity-typed places (restrooms, waiting areas, food stalls) ranked by walking distance from the origin node — one Dijkstra settle ranks them all. The origin is the current location's node id, the same `from` the route endpoint takes; unreachable amenities are absent, and no amenity at all is an empty list, never an error. Distances are authored SVG units, the unit NavigationRoute.totalDistance sums.
//	@Tags			navigation
//	@Produce		json
//	@Param			from	query	string	true	"Current location node id (e.g. I-1301/node-reception)"
//	@Param			limit	query	int		false	"Max amenities to return (default 12, max 50)"
//	@Param			accessibleOnly	query	boolean	false	"Rank by accessible routes only (stairs skipped, FR-20)"
//	@Success		200	{object}	navigation.AmenitySearch
//	@Failure		400	{object}	httpx.ErrorResponse	"missing from, limit out of range, or accessibleOnly is not a boolean"
//	@Failure		404	{object}	httpx.ErrorResponse	"unknown origin node"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/navigation/amenities [get]
func (h *Handler) amenities(c fiber.Ctx) error {
	from := c.Query("from")
	if from == "" {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "from is required"))
	}
	accessibleOnly, err := parseAccessibleOnly(c)
	if err != nil {
		return httpx.Error(c, err)
	}
	limit := 12
	if raw := c.Query("limit"); raw != "" {
		parsed, convErr := strconv.Atoi(raw)
		if convErr != nil || parsed < 1 || parsed > 50 {
			return httpx.Error(c, apperr.New(apperr.KindInvalid, "limit must be an integer between 1 and 50"))
		}
		limit = parsed
	}
	result, err := h.service.NearestAmenities(c.Context(), from, RouteOptions{AccessibleOnly: accessibleOnly}, limit)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(result)
}

// parseAccessibleOnly reads the shared routing flag. AccessibleOnly (#99):
// absent means the full graph; anything present must parse as a boolean
// (strconv's forms: true/false/1/0/t/f...), so a typo like "yes" is a 400
// rather than a silent full-graph route for a wheelchair user.
func parseAccessibleOnly(c fiber.Ctx) (bool, error) {
	if raw := c.Query("accessibleOnly"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return false, apperr.New(apperr.KindInvalid, "accessibleOnly must be a boolean")
		}
		return parsed, nil
	}
	return false, nil
}
