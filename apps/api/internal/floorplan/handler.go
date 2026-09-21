package floorplan

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/auth"
	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler serves the floor listing and the plan assets it points at.
//
// The split between the two routes is the caching design (ADR-0015): the
// listing is small and changes whenever a plan is replaced, so it
// revalidates; the plan is large and, because its URL names its own digest,
// can never change, so it is fetched once and never asked about again.
type Handler struct {
	plans  Service
	floors hospitalmap.Service
}

func NewHandler(plans Service, floors hospitalmap.Service) *Handler {
	return &Handler{plans: plans, floors: floors}
}

// Register mounts the public plan routes under the given /api/v1 router.
// Both are unguarded reads: a patient needs the plan of the floor they are
// standing on before they have claimed anything, and a plan is a drawing of
// a public building, not patient data.
func (h *Handler) Register(router fiber.Router, adminGuard fiber.Handler) {
	router.Get("/floors", h.listFloors)
	router.Get("/floors/:floorId/plan/:asset", h.plan)

	// Replacing a floor plan changes what every patient in the building
	// sees, so it is ADMIN-only — per route, never group-wide (ADR-0010).
	router.Post("/admin/floors/:floorId/plan", adminGuard, h.upload)
	router.Get("/admin/floors/:floorId/plans", adminGuard, h.listPlans)
	router.Post("/admin/floors/:floorId/plans/:planId/activate", adminGuard, h.activate)
	router.Get("/admin/floors/:floorId/plans/:planId/raw", adminGuard, h.raw)
	router.Get("/admin/floors/:floorId/health", adminGuard, h.health)
}

// FloorView is one floor and where to fetch its drawing. PlanURL is absent
// on a floor with no plan yet — the same explicit "not available" state the
// rest of the map model uses, rather than a URL that would 404.
type FloorView struct {
	FloorID    string  `json:"floorId"`
	BuildingID string  `json:"buildingId"`
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	LevelOrder int     `json:"levelOrder"`
	ViewBox    *string `json:"viewBox,omitempty"`
	PlanURL    *string `json:"planUrl,omitempty"`
	// ActivePlanID names the plan PlanURL was built from, so a client that
	// already has that plan's id (e.g. from the admin history listing) can
	// tell it is the active one directly, rather than reconstructing the
	// fact by checking whether its digest appears inside the URL.
	ActivePlanID *string `json:"activePlanId,omitempty"`
}

// listFloors godoc
//
//	@Summary		List floors and their plan URLs
//	@Description	#105/ADR-0015: every floor in display order with the URL of its current plan. This is the pointer document the web app revalidates; the plan URL it hands back is immutable and cached for a year. Replaces the floor list apps/web used to hardcode beside its bundled plan assets.
//	@Tags			floors
//	@Produce		json
//	@Success		200	{array}		floorplan.FloorView
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/floors [get]
func (h *Handler) listFloors(c fiber.Ctx) error {
	floors, err := h.floors.ListFloors(c.Context())
	if err != nil {
		return httpx.Error(c, err)
	}

	planIDs := make([]string, 0, len(floors))
	for _, floor := range floors {
		if floor.ActivePlanID != nil {
			planIDs = append(planIDs, *floor.ActivePlanID)
		}
	}
	digests, err := h.plans.Digests(c.Context(), planIDs)
	if err != nil {
		return httpx.Error(c, err)
	}

	views := make([]FloorView, 0, len(floors))
	for _, floor := range floors {
		view := FloorView{
			FloorID:    floor.ID,
			BuildingID: floor.BuildingID,
			Code:       floor.Code,
			Name:       floor.Name,
			LevelOrder: floor.LevelOrder,
			ViewBox:    floor.ViewBox,
		}
		if floor.ActivePlanID != nil {
			if sha, ok := digests[*floor.ActivePlanID]; ok {
				url := "/api/v1/floors/" + floor.ID + "/plan/" + sha + ".svg"
				view.PlanURL = &url
				view.ActivePlanID = floor.ActivePlanID
			}
		}
		views = append(views, view)
	}

	// The pointer moves whenever a plan is replaced, so this must be
	// revalidated rather than reused — it is small, and it is the only
	// thing standing between a new plan and the patients who need it.
	c.Set(fiber.HeaderCacheControl, "no-cache")
	return c.JSON(views)
}

// plan godoc
//
//	@Summary		Get a floor plan
//	@Description	#105/ADR-0015: the normalized SVG for a floor, addressed by its own sha256. The response is immutable — a different drawing is a different digest and therefore a different URL — so it is cached for a year and never revalidated. Take this URL from the floors listing rather than assembling it.
//	@Tags			floors
//	@Produce		image/svg+xml
//	@Param			floorId	path		string	true	"Floor id (e.g. I-1301)"
//	@Param			asset	path		string	true	"The plan's sha256 with an .svg suffix, as the floors listing spells it"
//	@Success		200		{string}	string	"the floor plan SVG"
//	@Failure		404		{object}	httpx.ErrorResponse	"no plan with that digest on that floor"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/floors/{floorId}/plan/{asset} [get]
func (h *Handler) plan(c fiber.Ctx) error {
	digest, ok := strings.CutSuffix(c.Params("asset"), ".svg")
	if !ok {
		return httpx.Error(c, ErrPlanNotFound)
	}
	svg, err := h.plans.SVG(c.Context(), c.Params("floorId"), digest)
	if err != nil {
		return httpx.Error(c, err)
	}

	c.Set(fiber.HeaderContentType, "image/svg+xml; charset=utf-8")
	// The digest is in the URL, so this body can never change under it.
	c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
	// This URL can be opened directly, and a directly-navigated SVG runs in
	// the API's own origin. The normalizer is what makes the content safe;
	// these two are the second line behind it, not a substitute for it.
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderContentSecurityPolicy, "default-src 'none'; style-src 'unsafe-inline'")
	return c.SendString(svg)
}

// upload godoc
//
//	@Summary		Upload a floor plan
//	@Description	FR-11/ADR-0015: replaces the floor's plan. Send the SVG as the raw request body. The upload is parsed, held to a closed allowlist and re-serialized before storage, so a rejection means the file has to change — nothing is silently stripped. A place or navigation node the plan does not draw is a warning rather than a rejection: the app already shows such a place as not routable yet. Plans are append-only, so this adds a row and moves the floor's pointer; re-uploading an unchanged file is idempotent.
//	@Tags			admin
//	@Security		bearerAuth
//	@Accept			image/svg+xml
//	@Produce		json
//	@Param			floorId	path		string	true	"Floor id (e.g. I-1301)"
//	@Success		201		{object}	floorplan.Stored
//	@Failure		400		{object}	httpx.ErrorResponse	"the plan broke a rule — the message names which"
//	@Failure		401		{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403		{object}	httpx.ErrorResponse	"authenticated, but not an ADMIN"
//	@Failure		404		{object}	httpx.ErrorResponse	"unknown floor"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/admin/floors/{floorId}/plan [post]
func (h *Handler) upload(c fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "floor plan: the request body is empty"))
	}
	// NFR-09's "who" is the acting user, resolved from the token — never
	// something the client sent along with the file.
	principal := auth.PrincipalFromContext(c.Context())
	stored, err := h.plans.Upload(c.Context(), c.Params("floorId"), body, principal.UserID)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(stored)
}

// listPlans godoc
//
//	@Summary		List a floor's plans
//	@Description	ADR-0015: every plan stored for this floor, newest first, with the warnings each was accepted with. Plans are never deleted, so this is both the history and the list a rollback picks from.
//	@Tags			admin
//	@Security		bearerAuth
//	@Produce		json
//	@Param			floorId	path		string	true	"Floor id"
//	@Success		200		{array}		floorplan.Stored
//	@Failure		401		{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403		{object}	httpx.ErrorResponse	"authenticated, but not an ADMIN"
//	@Failure		404		{object}	httpx.ErrorResponse	"unknown floor"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/admin/floors/{floorId}/plans [get]
func (h *Handler) listPlans(c fiber.Ctx) error {
	floorID := c.Params("floorId")
	// Resolve the floor first: ListPlans alone can't distinguish "unknown
	// floor" from "floor exists, no plans yet" — both answer an empty slice.
	if _, err := h.floors.GetFloor(c.Context(), floorID); err != nil {
		return httpx.Error(c, err)
	}
	plans, err := h.plans.ListPlans(c.Context(), floorID)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(plans)
}

// activate godoc
//
//	@Summary		Make a stored plan the floor's active one
//	@Description	ADR-0015: the rollback. Plans are append-only, so returning to an earlier drawing moves the floor's pointer rather than restoring anything.
//	@Tags			admin
//	@Security		bearerAuth
//	@Produce		json
//	@Param			floorId	path		string	true	"Floor id"
//	@Param			planId	path		string	true	"Plan id from the plans listing"
//	@Success		200		{object}	floorplan.Stored
//	@Failure		401		{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403		{object}	httpx.ErrorResponse	"authenticated, but not an ADMIN"
//	@Failure		404		{object}	httpx.ErrorResponse	"no such plan on that floor"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/admin/floors/{floorId}/plans/{planId}/activate [post]
func (h *Handler) activate(c fiber.Ctx) error {
	stored, err := h.plans.Activate(c.Context(), c.Params("floorId"), c.Params("planId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(stored)
}

// raw godoc
//
//	@Summary		Download a plan as it was uploaded
//	@Description	ADR-0015: the original bytes, before normalization, so an admin can get back the file they sent. This is never what a patient is served — that is always the normalized artifact.
//	@Tags			admin
//	@Security		bearerAuth
//	@Produce		octet-stream
//	@Param			floorId	path		string	true	"Floor id"
//	@Param			planId	path		string	true	"Plan id from the plans listing"
//	@Success		200		{string}	string	"the uploaded file"
//	@Failure		401		{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403		{object}	httpx.ErrorResponse	"authenticated, but not an ADMIN"
//	@Failure		404		{object}	httpx.ErrorResponse	"no such plan on that floor"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/admin/floors/{floorId}/plans/{planId}/raw [get]
func (h *Handler) raw(c fiber.Ctx) error {
	svg, err := h.plans.Raw(c.Context(), c.Params("floorId"), c.Params("planId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	// These bytes never went through the allowlist — this is the one copy in
	// the system that has not. Hand them over as a file to save rather than
	// as something a browser should render.
	c.Set(fiber.HeaderContentType, "application/octet-stream")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+c.Params("planId")+`.svg"`)
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	return c.SendString(svg)
}

// health godoc
//
//	@Summary		Re-check the floor's active plan against the map model
//	@Description	ADR-0015: recomputes the active plan's warnings against the map model as it stands right now, unlike the warnings on the plan itself (from /plans), which are a snapshot frozen at upload time. Empty when the floor has no active plan.
//	@Tags			admin
//	@Security		bearerAuth
//	@Produce		json
//	@Param			floorId	path		string	true	"Floor id"
//	@Success		200		{array}		floorplan.Warning
//	@Failure		401		{object}	httpx.ErrorResponse	"missing, malformed, or expired staff access token"
//	@Failure		403		{object}	httpx.ErrorResponse	"authenticated, but not an ADMIN"
//	@Failure		404		{object}	httpx.ErrorResponse	"unknown floor"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/admin/floors/{floorId}/health [get]
func (h *Handler) health(c fiber.Ctx) error {
	warnings, err := h.plans.Health(c.Context(), c.Params("floorId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(warnings)
}
