package visitlink

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/skip2/go-qrcode"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
	"carepath/apps/api/internal/session"
)

// Handler exposes the visit link over HTTP: a mint surface the HIS calls
// with its API key, and a redeem surface the patient web calls behind its
// session. This is the system's first inbound HIS→CarePath call (everything
// else is the outbound event feed) — the key header is not optional.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the visit-link routes. hisAPIKey gates the mint route's
// very registration (fail closed — an unset key must not leave a
// credential-minting endpoint open); sessionGuard is the patient session
// guard from the composition root, applied to redeem.
func (h *Handler) Register(router fiber.Router, hisAPIKey string, sessionGuard fiber.Handler) {
	router.Post("/his/visits/:visitId/patient-link", h.withHISKey(hisAPIKey, h.mint))
	router.Post("/journeys/claim", sessionGuard, h.redeem)
}

// withHISKey answers 401 before anything else runs when the caller cannot
// present the deployment's HIS API key. The compare is constant-time.
func (h *Handler) withHISKey(key string, next fiber.Handler) fiber.Handler {
	return func(c fiber.Ctx) error {
		if subtle.ConstantTimeCompare([]byte(c.Get("X-HIS-API-Key")), []byte(key)) != 1 {
			return httpx.Error(c, ErrBadKey)
		}
		return next(c)
	}
}

type redeemRequest struct {
	Token string `json:"token"`
}

type redeemResponse struct {
	VisitID string `json:"visitId"`
}

// mint godoc
//
//	@Summary		Mint the visit's patient link (HIS surface)
//	@Description	Returns the visit's navigation-slip link (#136): a high-entropy token behind /patient/journey#vt=… that the patient's session redeems into visit ownership. Repeat calls return the same URL (a re-print never invalidates the slip in the patient's hand); rotate=true discards it — lost slip, suspected leak — and the previous link stops redeeming. The visit's lifetime governs the link: ACTIVE is live, COMPLETED lives out VISIT_LINK_COMPLETED_GRACE, CANCELLED dies at once. Service-to-service: X-HIS-API-Key, the system's only inbound HIS→CarePath call. format=qr answers the same URL encoded as a PNG QR ready for the slip.
//	@Tags			his
//	@Security		hisApiKey
//	@Param			visitId	path	string	true	"Visit id (VN)"
//	@Param			format	query	string	false	"url (JSON, default) or qr (image/png of the same URL)"
//	@Param			rotate	query	boolean	false	"Discard the current token and mint a new one (default false)"
//	@Success		200	{object}	visitlink.Link	"format=url: the link as JSON"
//	@Success		200	{string}	bytestream	"format=qr: the link as a PNG QR"
//	@Failure		400	{object}	httpx.ErrorResponse	"unknown format value"
//	@Failure		401	{object}	httpx.ErrorResponse	"missing or invalid HIS API key"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit not projected"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/his/visits/{visitId}/patient-link [post]
func (h *Handler) mint(c fiber.Ctx) error {
	format := c.Query("format", "url")
	if format != "url" && format != "qr" {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "format must be url or qr"))
	}
	// c.Params is a zero-copy view into fasthttp's recycled request buffer —
	// the repo must never retain it past this request.
	visitID := strings.Clone(c.Params("visitId"))
	link, err := h.service.Mint(c.Context(), visitID, c.Query("rotate") == "true")
	if err != nil {
		return httpx.Error(c, err)
	}
	if format == "url" {
		return c.JSON(link)
	}
	png, err := qrcode.Encode(link.VisitURL, qrcode.Medium, 256)
	if err != nil {
		return httpx.Error(c, apperr.Wrapf(apperr.KindInternal, err, "visitlink: encode qr"))
	}
	c.Set(fiber.HeaderContentType, "image/png")
	return c.Send(png)
}

// redeem godoc
//
//	@Summary		Redeem a visit link token for this session
//	@Description	Exchanges the slip-held link token (#136) for the visit it addresses and records the #96 claim — the identity behind the patient session becomes the visit's owner, exactly as typing a VN used to do on demo deployments. The token rides the request body, never a URL. Unknown, rotated, cancelled, past-grace, and secret-rotated tokens all answer the journey read's 404.
//	@Tags			auth
//	@Security		bearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			request	body	redeemRequest	true	"the vt token from the slip URL's #vt= fragment"
//	@Success		200	{object}	redeemResponse
//	@Failure		401	{object}	httpx.ErrorResponse	"missing or invalid patient session"
//	@Failure		404	{object}	httpx.ErrorResponse	"visit unknown or link not redeemable — indistinguishable"
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/journeys/claim [post]
func (h *Handler) redeem(c fiber.Ctx) error {
	var req redeemRequest
	if err := c.Bind().Body(&req); err != nil {
		return httpx.Error(c, apperr.Wrapf(apperr.KindInvalid, err, "invalid request body"))
	}
	id, ok := session.IdentityFromCtx(c)
	if !ok {
		return httpx.Error(c, session.ErrNotFound)
	}
	visitID, err := h.service.Redeem(c.Context(), id, req.Token)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(redeemResponse{VisitID: visitID})
}
