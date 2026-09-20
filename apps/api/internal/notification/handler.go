package notification

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the notification module's patient surface: the opt-out
// toggle for a visit's queue-proximity notifications (#104).
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the notification routes under the given /api/v1 router.
// The patient guard (#96) is the same visit-claim boundary as the journey
// read — only the patient (or a share-less proxy of nobody else) may flip
// their own notifications.
func (h *Handler) Register(router fiber.Router, patientGuard fiber.Handler) {
	router.Get("/journeys/:visitId/notifications", patientGuard, h.get)
	router.Put("/journeys/:visitId/notifications", patientGuard, h.set)
}

// setRequestBody carries the toggle.
type setRequestBody struct {
	Enabled *bool `json:"enabled"`
}

// get godoc
//
//	@Summary		Read the visit's queue-notification preference
//	@Description	FR-21 (#104): whether queue-proximity notifications are enabled for this visit. No stored preference means enabled. Patient-surface (#96): the session's identity must have claimed the visit.
//	@Tags			notifications
//	@Security		bearerAuth
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Success		200	{object}	notification.Pref
//	@Failure		404	{object}	httpx.ErrorResponse	"unknown visit"
//	@Router			/api/v1/journeys/{visitId}/notifications [get]
func (h *Handler) get(c fiber.Ctx) error {
	pref, err := h.service.GetPref(c.Context(), c.Params("visitId"))
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(pref)
}

// set godoc
//
//	@Summary		Set the visit's queue-notification preference
//	@Description	FR-21 (#104): enable or disable queue-proximity notifications for this visit. The change takes effect on the next sweep. Patient-surface (#96): the session's identity must have claimed the visit.
//	@Tags			notifications
//	@Security		bearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			visitId	path	string	true	"Visit ID"
//	@Param			body	body	notification.setRequestBody	true	"The toggle"
//	@Success		200	{object}	notification.Pref
//	@Failure		400	{object}	httpx.ErrorResponse	"invalid body"
//	@Failure		404	{object}	httpx.ErrorResponse	"unknown visit"
//	@Router			/api/v1/journeys/{visitId}/notifications [put]
func (h *Handler) set(c fiber.Ctx) error {
	var body setRequestBody
	if err := json.Unmarshal(c.Body(), &body); err != nil || body.Enabled == nil {
		return httpx.Error(c, apperr.New(apperr.KindInvalid, "enabled must be a boolean"))
	}
	pref, err := h.service.SetPref(c.Context(), c.Params("visitId"), *body.Enabled)
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(pref)
}
