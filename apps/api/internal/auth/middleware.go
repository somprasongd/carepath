package auth

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// principalKey is unexported: only this package can put a principal in a
// context, so a handler reading one knows it was verified here.
type principalKey struct{}

// PrincipalFromContext returns the acting user the middleware resolved, or
// the zero Principal when the call is not behind RequireRole (system work).
func PrincipalFromContext(ctx context.Context) Principal {
	p, _ := ctx.Value(principalKey{}).(Principal)
	return p
}

// RequireRole guards a route group (ADR-0010 §8). With no roles it only
// authenticates; otherwise the principal must hold at least one of them.
// Unauthenticated is 401, authenticated-but-wrong-role is 403, both through
// the module's apperr values so httpx.Error logs them once at the boundary.
func RequireRole(s Service, roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if !ok || token == "" {
			return httpx.Error(c, ErrUnauthorized)
		}
		principal, err := s.ParseAccessToken(token)
		if err != nil {
			return httpx.Error(c, err)
		}
		if len(roles) > 0 && !principal.HasRole(roles...) {
			return httpx.Error(c, ErrForbidden)
		}
		c.SetContext(context.WithValue(c.Context(), principalKey{}, principal))
		return c.Next()
	}
}
