package session

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the session module over HTTP.
type Handler struct {
	service Service
	claims  ClaimRepo
}

func NewHandler(service Service, claims ClaimRepo) *Handler {
	return &Handler{service: service, claims: claims}
}

// Register mounts the session routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Post("/auth/session", h.create)
	router.Get("/auth/session", h.get)
	// The visit claim (#96) is the bridge from a verified identity to the
	// visits it may read — the only way ownership is minted. Session-guarded,
	// not visit-guarded: the claim itself is the act of claiming.
	router.Post("/journeys/:visitId/claim", RequireSession(h.service), h.claim)
}

type createRequest struct {
	Source  string `json:"source"`
	IDToken string `json:"idToken"`
}

type identityResponse struct {
	Source      string `json:"source"`
	ExternalID  string `json:"externalId"`
	DisplayName string `json:"displayName,omitempty"`
}

type createResponse struct {
	SessionToken string           `json:"sessionToken"`
	Identity     identityResponse `json:"identity"`
	ExpiresAt    time.Time        `json:"expiresAt"`
}

// create godoc
//
//	@Summary		Create a patient session
//	@Description	Exchanges a LINE ID token (or, when enabled, a demo identity) for an opaque CarePath session token.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		createRequest	true	"source: line|demo, idToken: the LINE ID token (ignored for demo)"
//	@Success		200		{object}	createResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"invalid request body"
//	@Failure		401		{object}	httpx.ErrorResponse	"invalid identity token, or demo auth disabled"
//	@Router			/api/v1/auth/session [post]
func (h *Handler) create(c fiber.Ctx) error {
	var req createRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Error(c, apperr.Wrapf(apperr.KindInvalid, err, "invalid request body"))
	}

	sess, err := h.service.Create(c.Context(), req.Source, req.IDToken)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(createResponse{
		SessionToken: sess.Token,
		Identity: identityResponse{
			Source:      sess.Identity.Source,
			ExternalID:  sess.Identity.ExternalID,
			DisplayName: sess.Identity.DisplayName,
		},
		ExpiresAt: sess.ExpiresAt,
	})
}

// get godoc
//
//	@Summary		Get the current session identity
//	@Description	Resolves the Authorization: Bearer session token to the identity behind it.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	identityResponse
//	@Failure		401	{object}	httpx.ErrorResponse	"session not found or expired"
//	@Router			/api/v1/auth/session [get]
func (h *Handler) get(c fiber.Ctx) error {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok || token == "" {
		return httpx.Error(c, ErrNotFound)
	}

	id, err := h.service.Get(c.Context(), token)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(identityResponse{
		Source:      id.Source,
		ExternalID:  id.ExternalID,
		DisplayName: id.DisplayName,
	})
}

// claim godoc
//
//	@Summary		Claim a visit for this session's identity
//	@Description	Records that the identity behind the patient session owns the visit (#96) — the only way read access to a journey is minted. Knowing the VN is the credential, matching the product's patient front door. Idempotent: re-claiming a visit already owned by this identity succeeds. A visit that was never projected answers 404, indistinguishable from the journey read's unknown-visit 404.
//	@Tags			auth
//	@Security		bearerAuth
//	@Param			visitId	path	string	true	"Visit id"
//	@Success		204	"No content"
//	@Failure		401	{object}	httpx.ErrorResponse	"missing or invalid patient session"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit not projected"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/{visitId}/claim [post]
func (h *Handler) claim(c fiber.Ctx) error {
	id, _ := c.Locals(identityLocalKey).(identity.Identity)
	if err := h.claims.Claim(c.Context(), id, c.Params("visitId")); err != nil {
		return httpx.Error(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
