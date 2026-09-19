package logger

import (
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Middleware returns a Fiber middleware that assigns a request ID, builds a
// request-scoped logger, stores it in the request context (reachable by any
// handler via FromContext), recovers panics, and logs one line per completed
// request.
func Middleware(base *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		requestID := newRequestID(c.Get("X-Request-ID"))
		log := RequestLogger(base, c.Method(), c.Path(), requestID)
		c.Set("X-Request-ID", requestID)
		c.SetContext(IntoContext(c.Context(), log))

		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = fiber.ErrInternalServerError
			}
		}()

		err = c.Next()
		if err != nil {
			// Handler returned an unhandled error; the Fiber error handler
			// renders the response after the middleware chain unwinds.
			log.Error("request handler error",
				"error", err.Error(),
				"duration_ms", elapsedMS(time.Since(start)),
			)
			return err
		}

		status := c.Response().StatusCode()
		log.Info("request completed",
			"status", status,
			"duration_ms", elapsedMS(time.Since(start)),
		)
		return nil
	}
}
