// Package httpx holds HTTP transport helpers shared by every module's
// handlers. Error is the single place that maps a domain error to its HTTP
// status, writes the response envelope, and logs the failure.
package httpx

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/logger"
)

// ErrorResponse is the error envelope returned by all API endpoints. Code
// is the machine-readable apperr kind ("invalid", "not_found", …) clients
// branch on (#28); Error is the human-safe message.
type ErrorResponse struct {
	Error string `json:"error" example:"visit not found"`
	Code  string `json:"code" example:"not_found"`
}

// Error maps err to a status via apperr.KindOf, logs it with the
// request-scoped logger, and writes the error envelope. Every 5xx kind is
// masked: both KindInternal and KindUpstream wrap a raw underlying cause
// (a driver error, a dial failure) that is unsafe to expose, so the client
// gets a generic, kind-specific message while the cause stays in the logs.
func Error(c fiber.Ctx, err error) error {
	kind := apperr.KindOf(err)
	status := kind.HTTPStatus()
	log := logger.FromContext(c.Context())

	if status >= 500 {
		log.Error("request error", "kind", kind.String(), "error", err.Error(), "status", status)
		return c.Status(status).JSON(ErrorResponse{Error: genericMessage(kind), Code: kind.String()})
	}

	log.Warn("request error", "kind", kind.String(), "error", err.Error(), "status", status)
	return c.Status(status).JSON(ErrorResponse{Error: err.Error(), Code: kind.String()})
}

func genericMessage(kind apperr.Kind) string {
	if kind == apperr.KindUpstream {
		return "upstream service unavailable"
	}
	return "internal server error"
}
