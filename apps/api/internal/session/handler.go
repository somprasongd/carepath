package session

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the session module over HTTP.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the session routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Post("/auth/session", h.create)
	router.Get("/auth/session", h.get)
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
