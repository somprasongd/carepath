package journey

import (
	"github.com/gofiber/fiber/v3"

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
	router.Post("/journeys/:visitId/steps/:stepKey/transition", h.transitionStep)
	router.Post("/journeys/:visitId/clinics/:clinicCode/close-round", h.closeRound)
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
//	@Description	Returns the CarePath-derived journey plan (ADR-0009): steps in display order, each resolved to its service point, with every currently-actionable step and CarePath's recommendation among them. A completed visit is reported with completed=true and no actionable steps.
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
//	@Description	Every projected journey, freshest sync first — the same per-visit shape as the single-journey read. Reads the CarePath projection only; a visit not yet ingested is absent until its first fact lands.
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
//	@Summary		Transition one step
//	@Description	Staff command that sets one step's status directly (ADR-0009 — CarePath owns step status; never forwarded to the HIS) and returns the refreshed journey with the plan recomputed. commandId is an optional idempotency key; source names the acting surface for the audit trail.
//	@Tags			journeys
//	@Accept			json
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Param			stepKey	path	string	true	"Step key"
//	@Param			body	body	journey.transitionRequestBody	true	"Transition command"
//	@Success		200	{object}	journey.View
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid body or unknown target status"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit or step not found, or journey not projected"
//	@Failure		409	{object}	httpx.ErrorResponse	"illegal transition"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Failure		502	{object}	httpx.ErrorResponse	"upstream HIS error"
//	@Router			/api/v1/journeys/{visitId}/steps/{stepKey}/transition [post]
func (h *Handler) transitionStep(c fiber.Ctx) error {
	var body transitionRequestBody
	if err := c.Bind().Body(&body); err != nil {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "invalid request body"))
	}
	stepKey := c.Params("stepKey")
	if stepKey == "" {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "stepKey is required"))
	}
	view, err := h.service.TransitionStep(c.Context(), c.Params("visitId"), stepKey,
		TransitionCommand{CommandID: body.CommandID, To: body.To}, body.Source)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(view)
}

// closeRound godoc
//
//	@Summary		Close a clinic round
//	@Description	Staff override (ADR-0009 §4): confirms the visit's latest round at this clinic is finished, dropping any not-yet-started inferred return, without waiting for the HIS's encounter.completed fact.
//	@Tags			journeys
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Param			clinicCode	path	string	true	"Clinic code"
//	@Success		200	{object}	journey.View
//	@Failure		404	{object}	httpx.ErrorResponse	"visit not found, journey not projected, or no open round at this clinic"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/{visitId}/clinics/{clinicCode}/close-round [post]
func (h *Handler) closeRound(c fiber.Ctx) error {
	view, err := h.service.CloseRound(c.Context(), c.Params("visitId"), c.Params("clinicCode"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(view)
}
