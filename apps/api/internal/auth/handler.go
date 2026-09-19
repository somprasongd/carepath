package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the auth module over HTTP.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the public auth routes (login, refresh, logout) under the
// given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Post("/auth/login", h.login)
	router.Post("/auth/refresh", h.refresh)
	router.Post("/auth/logout", h.logout)
}

// RegisterMe mounts GET /auth/me with a per-route guard. The guard is passed
// in (not a guarded group) because fiber Group middleware is prefix-wide: a
// guarded /api/v1 group would seal every patient route behind staff auth.
func (h *Handler) RegisterMe(router fiber.Router, guard fiber.Handler) {
	router.Get("/auth/me", guard, h.me)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type identityResponse struct {
	UserID      string   `json:"userId"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
}

type tokensResponse struct {
	AccessToken      string           `json:"accessToken"`
	RefreshToken     string           `json:"refreshToken"`
	ExpiresAt        time.Time        `json:"expiresAt"`
	RefreshExpiresAt time.Time        `json:"refreshExpiresAt"`
	Identity         identityResponse `json:"identity"`
}

// login godoc
//
//	@Summary		Staff/admin login
//	@Description	Verifies a username against an argon2id password hash and returns a short-lived JWT access token plus a rotating opaque refresh token. Unknown username, wrong password, and deactivated account are deliberately indistinguishable (same 401 body, same timing).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		loginRequest	true	"username and password"
//	@Success		200		{object}	tokensResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"invalid request body"
//	@Failure		401		{object}	httpx.ErrorResponse	"invalid credentials"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/auth/login [post]
func (h *Handler) login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Error(c, apperr.Wrapf(apperr.KindInvalid, err, "invalid request body"))
	}
	pair, err := h.service.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(tokensResponseOf(pair))
}

// refresh godoc
//
//	@Summary		Refresh the token pair
//	@Description	Exchanges a refresh token for a new access + refresh pair; the presented token is spent. Presenting an already-spent token is treated as a leak: every refresh token of that user is revoked and the call returns 401.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		refreshRequest	true	"the refresh token issued by login or a previous refresh"
//	@Success		200		{object}	tokensResponse
//	@Failure		400		{object}	httpx.ErrorResponse	"invalid request body"
//	@Failure		401		{object}	httpx.ErrorResponse	"unknown, expired, revoked, or already-spent refresh token"
//	@Failure		500		{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/auth/refresh [post]
func (h *Handler) refresh(c fiber.Ctx) error {
	var req refreshRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Error(c, apperr.Wrapf(apperr.KindInvalid, err, "invalid request body"))
	}
	pair, err := h.service.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(tokensResponseOf(pair))
}

// logout godoc
//
//	@Summary		Log out
//	@Description	Revokes the presented refresh token. Idempotent — an unknown, expired or already-revoked token also returns 204, so a client can always clear its own state.
//	@Tags			auth
//	@Accept			json
//	@Success		204	"No content"
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid request body"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/auth/logout [post]
func (h *Handler) logout(c fiber.Ctx) error {
	var req refreshRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Error(c, apperr.Wrapf(apperr.KindInvalid, err, "invalid request body"))
	}
	if err := h.service.Logout(c.Context(), req.RefreshToken); err != nil {
		return httpx.Error(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// me godoc
//
//	@Summary		The current staff identity
//	@Description	The staff/admin identity and role codes behind the presented access token. The console calls it on boot to restore a session from a stored refresh token.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	identityResponse
//	@Failure		401	{object}	httpx.ErrorResponse	"missing, malformed, or expired access token"
//	@Router			/api/v1/auth/me [get]
func (h *Handler) me(c fiber.Ctx) error {
	p := PrincipalFromContext(c.Context())
	return c.JSON(identityResponse{
		UserID:      p.UserID,
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Roles:       p.Roles,
	})
}

func tokensResponseOf(pair TokenPair) tokensResponse {
	roles := pair.Identity.Roles
	if roles == nil {
		roles = []string{}
	}
	return tokensResponse{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		ExpiresAt:        pair.ExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
		Identity: identityResponse{
			UserID:      pair.Identity.UserID,
			Username:    pair.Identity.Username,
			DisplayName: pair.Identity.DisplayName,
			Roles:       roles,
		},
	}
}
