package session

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// RequireSession guards a route with the patient session (the counterpart of
// auth.RequireRole for the other identity kind, ADR-0010/0011). It only
// authenticates — a session stands for a patient identity, not a role. A
// staff JWT or share token presented here fails: it is simply not a session
// token, and the failure is the module's one 401.
func RequireSession(s Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if !ok || token == "" {
			return httpx.Error(c, ErrNotFound)
		}
		if _, err := s.Get(c.Context(), token); err != nil {
			return httpx.Error(c, err)
		}
		return c.Next()
	}
}
