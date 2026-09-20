package journey

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/auth"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
	"carepath/apps/api/internal/servicepoint"
)

// Handler exposes the journey module over HTTP.
type Handler struct {
	service Service
	// servicePoints answers the station-queue authorization question
	// ("may this user work this point", #102) — read via the module's
	// Service interface, never its repo.
	servicePoints servicepoint.Service
}

func NewHandler(service Service, servicePoints servicepoint.Service) *Handler {
	return &Handler{service: service, servicePoints: servicePoints}
}

// Register mounts the journey routes under the given /api/v1 router. The
// staff-only routes (the monitor read and both staff commands) take the
// staff guard; the patient journey read takes the patient guard (#96 — the
// session must have claimed the visit). Both guards are injected so the
// composition root decides the policy (ADR-0010).
func (h *Handler) Register(router fiber.Router, staffGuard, patientGuard fiber.Handler) {
	router.Get("/journeys/:visitId", patientGuard, h.getJourney)
	router.Get("/journeys/:visitId/queue", patientGuard, h.getQueue)
	router.Get("/staff/visits", staffGuard, h.listVisits)
	router.Get("/staff/planning-rules", staffGuard, h.planningRules)
	router.Get("/staff/queue/:servicePointId", staffGuard, h.getStationQueue)
	router.Post("/journeys/:visitId/steps/:stepKey/transition", staffGuard, h.transitionStep)
	router.Post("/journeys/:visitId/clinics/:clinicCode/close-round", staffGuard, h.closeRound)
}

// getStationQueue godoc
//
//	@Summary		Get one service point's working queue
//	@Description	FR-15/#102: who is being served (STARTED) and who is waiting (READY, longest-waiting first) at the service point, with patient detail and arrival/call times from the timeline. Patient-level data on the staff surface only (NFR-03) — never analytics. STAFF sees only points they are assigned to (user_service_point); ADMIN may read any point. Not assigned reads as 403, including a point id that does not exist.
//	@Tags			staff
//	@Security		bearerAuth
//	@Produce		json
//	@Param			servicePointId	path	string	true	"Service point ID (from /staff/my/service-points)"
//	@Success		200	{object}	journey.StationQueue
//	@Failure		401	{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403	{object}	httpx.ErrorResponse	"authenticated, but not assigned to this service point"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/staff/queue/{servicePointId} [get]
func (h *Handler) getStationQueue(c fiber.Ctx) error {
	p := auth.PrincipalFromContext(c.Context())
	if p.UserID == "" {
		// The staff guard resolves the principal before handlers run;
		// an empty one means the route was mounted without it.
		return httpx.Error(c, apperr.New(apperr.KindUnauthorized, "missing principal"))
	}
	if !p.HasRole(auth.RoleAdmin) {
		assigned, err := h.servicePoints.IsAssigned(c.Context(), p.UserID, c.Params("servicePointId"))
		if err != nil {
			return httpx.Error(c, err)
		}
		if !assigned {
			return httpx.Error(c, apperr.New(apperr.KindForbidden,
				"you are not assigned to this service point"))
		}
	}
	queue, err := h.service.GetStationQueue(c.Context(), c.Params("servicePointId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(queue)
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
//	@Description	Returns the CarePath-derived journey plan (ADR-0009): steps in display order, each resolved to its service point, with every currently-actionable step and CarePath's recommendation among them. A completed visit is reported with completed=true and no actionable steps. Patient-surface (#96): the patient session's identity must have claimed this visit; an unclaimed or unknown visit answers the same 404.
//	@Tags			journeys
//	@Security		bearerAuth
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Success		200	{object}	journey.View
//	@Failure		401	{object}	httpx.ErrorResponse	"missing or invalid patient session"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit unknown, or not claimed by this session — indistinguishable"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/{visitId} [get]
func (h *Handler) getJourney(c fiber.Ctx) error {
	view, err := h.service.GetJourney(c.Context(), c.Params("visitId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(view)
}

// getQueue godoc
//
//	@Summary		Get the patient queue picture for the next steps
//	@Description	FR-17: for every currently-actionable step bound to a service point, how many people are waiting ahead of this patient there right now, and that point's average experienced wait so far today. Averages over zero samples are null, not 0 — "no data yet" and "zero minutes" are different facts, and the client must not render a guess. estimatedWaitMinutes is the naive product of the two, null whenever the average is null. Patient-surface (#96): the session's identity must have claimed this visit; an unclaimed or unknown visit answers the same 404 as the journey read.
//	@Tags			journeys
//	@Security		bearerAuth
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Success		200	{object}	journey.QueueView
//	@Failure		401	{object}	httpx.ErrorResponse	"missing or invalid patient session"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit unknown, or not claimed by this session — indistinguishable"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/{visitId}/queue [get]
func (h *Handler) getQueue(c fiber.Ctx) error {
	view, err := h.service.GetQueue(c.Context(), c.Params("visitId"))
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

// planningRules godoc
//
//	@Summary		Get the journey planning rules
//	@Description	FR-13/US-16: the rules the planner actually runs on — the phase ladder (ADR-0009 §3), the order-type → step-kind mapping, and worked examples computed by the real Plan function. Read-only: editing rules is a code change, not a request. Codes only (ADR-0012); display text lives in the client.
//	@Tags			staff
//	@Security		bearerAuth
//	@Produce		json
//	@Success		200	{object}	journey.PlanningRules
//	@Failure		401	{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403	{object}	httpx.ErrorResponse	"authenticated, but the user's roles do not allow this action"
//	@Router			/api/v1/staff/planning-rules [get]
func (h *Handler) planningRules(c fiber.Ctx) error {
	// Derived purely from planner.go (constants, map, Plan) — no database or
	// HIS access, so the descriptor is served straight from the function.
	return c.JSON(PlanningRulesDescriptor())
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
	p := auth.PrincipalFromContext(c.Context())
	view, err := h.service.TransitionStep(c.Context(), c.Params("visitId"), stepKey,
		TransitionCommand{CommandID: body.CommandID, To: body.To}, body.Source,
		Actor{UserID: p.UserID, Username: p.Username})
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
	// The close-round body is empty, so the surface is named here rather
	// than carried in the request like a transition's source field.
	p := auth.PrincipalFromContext(c.Context())
	view, err := h.service.CloseRound(c.Context(), c.Params("visitId"), c.Params("clinicCode"),
		"staff-web", Actor{UserID: p.UserID, Username: p.Username})
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(view)
}
