package location

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the location module over HTTP: reporting a scanned fix
// and reading the visit's current location.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the location routes under the given /api/v1 router,
// alongside the other per-visit journey endpoints (the #43 auth guard is
// meant to cover this group too).
func (h *Handler) Register(router fiber.Router) {
	router.Post("/journeys/:visitId/location", h.report)
	router.Get("/journeys/:visitId/location", h.current)
}

// RegisterDemo mounts the demo-only simulator routes under the given
// /api/v1 router (#33): POST /demo/zigbee/location stands in for the
// external positioning service during demos. It is separate from Register
// so the canonical per-visit routes and the demo surface can be guarded
// independently once #43 lands.
func (h *Handler) RegisterDemo(router fiber.Router) {
	router.Post("/demo/zigbee/location", h.simulateZigbee)
}

// reportRequestBody is the client-facing scan result. Raw is the scanned
// payload exactly as read — the server resolves it, the client never
// interprets it.
type reportRequestBody struct {
	Source string `json:"source"`
	Raw    string `json:"raw"`
}

// report godoc
//
//	@Summary		Report a scanned location fix
//	@Description	Records a location fix for the visit and makes it the current location. The raw payload is the scanned string (a CarePath location QR carries a place reference only — never patient data); the server resolves it through the matching provider to a canonical navigation node and returns the recorded observation.
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Param			body	body	location.reportRequestBody	true	"Scanned fix: source (e.g. QR) and raw payload"
//	@Success		200	{object}	location.Observation
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid body, unknown source, or a fix that does not resolve to a known location"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/{visitId}/location [post]
func (h *Handler) report(c fiber.Ctx) error {
	var body reportRequestBody
	if err := c.Bind().Body(&body); err != nil {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "invalid request body"))
	}
	obs, err := h.service.Report(c.Context(), c.Params("visitId"), Source(body.Source), body.Raw)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(obs)
}

// current godoc
//
//	@Summary		Get the visit's current location
//	@Description	The latest recorded location observation of the visit (QR scan, Zigbee fix, manual pick) — the canonical start point for routing.
//	@Tags			location
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Success		200	{object}	location.Observation
//	@Failure		404	{object}	httpx.ErrorResponse	"no location recorded for this visit yet"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/{visitId}/location [get]
func (h *Handler) current(c fiber.Ctx) error {
	obs, err := h.service.Current(c.Context(), c.Params("visitId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(obs)
}

// zigbeeSimulatorRequestBody is the demo operator's input: which visit the
// simulated tag is on, and the zone fix to report.
type zigbeeSimulatorRequestBody struct {
	VisitID    string   `json:"visitId"`
	FloorID    string   `json:"floorId"`
	Zone       string   `json:"zone"`
	Confidence *float64 `json:"confidence,omitempty"`
}

// simulateZigbee godoc
//
//	@Summary		Simulate a Zigbee zone fix
//	@Description	Demo-only simulator for the Zigbee positioning service (#33): reports a zone-level fix for the visit through the canonical ZIGBEE location provider and returns the recorded observation. The zone resolves to its representative navigation node, which becomes the routing start point like any other location. A real Zigbee integration will consume positioning-service pushes through a separate adapter, not this endpoint.
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Param			body	body	location.zigbeeSimulatorRequestBody	true	"Simulated zone fix: visitId, floorId, zone, optional confidence"
//	@Success		200	{object}	location.Observation
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid body, or a floor/zone that does not resolve to a known location"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/demo/zigbee/location [post]
func (h *Handler) simulateZigbee(c fiber.Ctx) error {
	var body zigbeeSimulatorRequestBody
	if err := c.Bind().Body(&body); err != nil {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "invalid request body"))
	}
	// The canonical entry point takes the provider's raw fix; the simulator
	// composes the same JSON message a positioning service would send (the
	// zigbee.Fix shape, mirrored here because importing the provider from
	// this package would create a cycle — the handler tests pin the two
	// together).
	raw, err := json.Marshal(struct {
		FloorID    string   `json:"floorId"`
		Zone       string   `json:"zone"`
		Confidence *float64 `json:"confidence,omitempty"`
	}{body.FloorID, body.Zone, body.Confidence})
	if err != nil {
		return httpx.Error(c, apperr.Wrapf(apperr.KindInternal, err, "location: compose zigbee fix"))
	}
	obs, err := h.service.Report(c.Context(), body.VisitID, SourceZigbee, string(raw))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(obs)
}
