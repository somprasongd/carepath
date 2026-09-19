// Package httpx holds HTTP transport helpers shared by every module's
// handlers. Error is the single place that maps a domain error to its HTTP
// status, writes the response envelope, and logs the failure.
package httpx

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/logger"
)

// ErrorResponse is the error envelope returned by all API endpoints.
type ErrorResponse struct {
	Error string `json:"error" example:"visit not found"`
}

// Error maps err to a status via apperr.KindOf, logs it with the
// request-scoped logger, and writes the error envelope. Internal failures
// are masked: the client receives a generic message while the cause stays in
// the logs.
func Error(c fiber.Ctx, err error) error {
	kind := apperr.KindOf(err)
	status := kind.HTTPStatus()
	log := logger.FromContext(c.Context())

	if kind == apperr.KindInternal {
		log.Error("request error", "kind", kind.String(), "error", err.Error(), "status", status)
		return c.Status(status).JSON(ErrorResponse{Error: "internal server error"})
	}

	if status >= 500 {
		log.Error("request error", "kind", kind.String(), "error", err.Error(), "status", status)
	} else {
		log.Warn("request error", "kind", kind.String(), "error", err.Error(), "status", status)
	}
	return c.Status(status).JSON(ErrorResponse{Error: err.Error()})
}
