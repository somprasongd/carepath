package share

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the share module over HTTP.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the share routes under the given /api/v1 router. The two
// management routes take the patient-session guard from the caller (per-route
// composition in the composition root, like journey.Register's staffGuard);
// the shared read is guarded by the share token itself and by nothing else.
func (h *Handler) Register(router fiber.Router, patientGuard fiber.Handler) {
	router.Post("/journeys/:visitId/share", patientGuard, h.create)
	router.Delete("/journeys/:visitId/share", patientGuard, h.revoke)
	router.Get("/shared/journey", h.getShared)
}

// create godoc
//
//	@Summary		Create a relative share link for a visit
//	@Description	Mints a read-only, time-limited link token (ADR-0011). Returns the raw token exactly once — store sha256 only — plus its expiry. The response carries no URL: the web app composes /shared#<token> itself so the API never needs its own origin. At most 5 active links per visit (409 beyond).
//	@Tags			share
//	@Security		bearerAuth
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit id"
//	@Success		200		{object}	LinkSecret
//	@Failure		401		{object}	httpx.ErrorResponse	"missing or invalid patient session"
//	@Failure		404		{object}	httpx.ErrorResponse	"visit not projected"
//	@Failure		409		{object}	httpx.ErrorResponse	"active-link cap reached"
//	@Router			/api/v1/journeys/{visitId}/share [post]
func (h *Handler) create(c fiber.Ctx) error {
	secret, err := h.service.CreateLink(c.Context(), c.Params("visitId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(secret)
}

// revoke godoc
//
//	@Summary		Stop sharing a visit ("หยุดแชร์")
//	@Description	Revokes every still-active share link of the visit. Idempotent — 204 whether or not any link existed.
//	@Tags			share
//	@Security		bearerAuth
//	@Param			visitId	path	string	true	"Visit id"
//	@Success		204		"No content"
//	@Failure		401		{object}	httpx.ErrorResponse	"missing or invalid patient session"
//	@Router			/api/v1/journeys/{visitId}/share [delete]
func (h *Handler) revoke(c fiber.Ctx) error {
	if err := h.service.RevokeAll(c.Context(), c.Params("visitId")); err != nil {
		return httpx.Error(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// getShared godoc
//
//	@Summary		Read the journey a share link points to
//	@Description	Resolves the share token (Authorization: Bearer) to the redacted SharedJourney — current step as Thai text, coarse status, service point name and floor, nothing else (ADR-0011 §3). Unknown, expired, and revoked tokens all answer with the same 401.
//	@Tags			share
//	@Security		shareAuth
//	@Produce		json
//	@Success		200	{object}	SharedJourney
//	@Failure		401		{object}	httpx.ErrorResponse	"invalid, expired, or revoked share token"
//	@Router			/api/v1/shared/journey [get]
func (h *Handler) getShared(c fiber.Ctx) error {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok || token == "" {
		return httpx.Error(c, ErrInvalidLink)
	}
	shared, err := h.service.Resolve(c.Context(), token)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(shared)
}
