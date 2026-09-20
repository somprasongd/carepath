package session

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/httpx"
)

// RequireSession guards a route with the patient session (the counterpart of
// auth.RequireRole for the other identity kind, ADR-0010/0011). It only
// authenticates — a session stands for a patient identity, not a role. A
// staff JWT or share token presented here fails: it is simply not a session
// token, and the failure is the module's one 401.
func RequireSession(s Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := identityFromRequest(c, s)
		if err != nil {
			return httpx.Error(c, err)
		}
		c.Locals(identityLocalKey, id)
		return c.Next()
	}
}

// RequirePatientVisit guards a visit-scoped patient route (#96): it resolves
// the session's identity and then requires a claim on the addressed visit.
// Unauthenticated is the module's 401; an identity without a claim answers
// ErrNotClaimed — the same 404 the journey read gives for an unknown visit,
// so ownership never leaks existence.
func RequirePatientVisit(s Service, claims ClaimRepo) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := identityFromRequest(c, s)
		if err != nil {
			return httpx.Error(c, err)
		}
		ok, err := claims.HasClaimed(c.Context(), id, c.Params("visitId"))
		if err != nil {
			return httpx.Error(c, err)
		}
		if !ok {
			return httpx.Error(c, ErrNotClaimed)
		}
		return c.Next()
	}
}

// identityLocalKey is set by RequireSession so a chained handler can reuse
// the resolved identity without re-reading the bearer.
const identityLocalKey = "session.identity"

// identityFromRequest resolves the Authorization bearer to the identity
// behind it, or the module's one 401.
func identityFromRequest(c fiber.Ctx, s Service) (identity.Identity, error) {
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if !ok || token == "" {
		return identity.Identity{}, ErrNotFound
	}
	return s.Get(c.Context(), token)
}
