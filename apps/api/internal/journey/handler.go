package journey

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
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
	// Staff-facing reads live under /staff so the NFR-08 auth guard (#43)
	// can cover the whole group.
	router.Get("/staff/visits", h.listVisits)
	router.Post("/journeys/:visitId/steps/:sequence/transition", h.transitionStep)
}

// transitionRequestBody is the client-facing command. CommandID is optional —
// the server assigns a fresh idempotency key when absent; Source names where
// the action came from (e.g. staff-web) for the audit trail.
type transitionRequestBody struct {
	To        string `json:"to"`
	CommandID string `json:"commandId,omitempty"`
	Source    string `json:"source,omitempty"`
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

// listVisits godoc
//
//	@Summary		List visits for the staff monitor
//	@Description	Every projected journey, freshest sync first — the same per-visit shape as the single-journey read (ordered steps, resolved service points, deterministic current/next). Reads the CarePath projection only; a visit not yet ingested is absent until its first event lands.
//	@Tags			staff
//	@Produce		json
//	@Success		200	{array}	journey.View
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/staff/visits [get]
func (h *Handler) listVisits(c fiber.Ctx) error {
	views, err := h.service.ListJourneys(c.Context())
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(views)
}

// transitionStep godoc
//
//	@Summary		Transition one service step
//	@Description	Forwards a step-status command to the HIS (the system of record, ADR-0008) and returns the refreshed journey with the recalculated next step. Illegal or out-of-order transitions are rejected by the HIS with 409. commandId is an optional idempotency key; source names the acting surface for the audit trail.
//	@Tags			journeys
//	@Accept			json
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Param			sequence	path	int	true	"Step sequence"
//	@Param			body	body	journey.transitionRequestBody	true	"Transition command"
//	@Success		200	{object}	journey.View
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid body or unknown target status"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit or step not found, or journey not projected"
//	@Failure		409	{object}	httpx.ErrorResponse	"illegal transition"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Failure		502	{object}	httpx.ErrorResponse	"upstream HIS error"
//	@Router			/api/v1/journeys/{visitId}/steps/{sequence}/transition [post]
func (h *Handler) transitionStep(c fiber.Ctx) error {
	var body transitionRequestBody
	if err := c.Bind().Body(&body); err != nil {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "invalid request body"))
	}
	sequence, err := strconv.Atoi(c.Params("sequence"))
	if err != nil || sequence < 1 {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "sequence must be a positive integer"))
	}
	view, err := h.service.TransitionStep(c.Context(), c.Params("visitId"), sequence,
		his.TransitionCommand{CommandID: body.CommandID, To: body.To}, body.Source)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(view)
}
