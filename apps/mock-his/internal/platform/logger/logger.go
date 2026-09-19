// Package logger provides the Mock HIS slog setup and a Fiber middleware
// that creates a request-scoped logger (with request ID and request
// attributes) and carries it in the request context. It deliberately mirrors
// apps/api's internal/platform/logger so both services emit the same log
// shape in docker compose logs — the duplication is the price of keeping the
// mock independent of CarePath code (it is swapped for a real HIS per
// ADR-0005, and apps/api's internal packages are not importable across
// modules anyway).
package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"
	"time"
)

type ctxKey struct{}

// New builds the base logger from the environment: LOG_LEVEL (debug|info|
// warn|error, default info) and LOG_FORMAT (json|text, default json).
func New() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handlerOpts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.ToLower(os.Getenv("LOG_FORMAT")) == "text" {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}
	return slog.New(handler)
}

// IntoContext returns a ctx carrying the given logger.
func IntoContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// FromContext returns the logger carried by ctx, falling back to the process
// default so callers never receive nil.
func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}

// newRequestID returns a random hex ID, honoring an incoming X-Request-ID so
// an upstream proxy can correlate logs later.
func newRequestID(incoming string) string {
	if incoming != "" {
		return incoming
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "req-unknown"
	}
	return hex.EncodeToString(buf)
}

// RequestLogger builds one request-scoped logger per request. It is exported
// for tests of the middleware; handlers should use FromContext.
func RequestLogger(base *slog.Logger, method, path, requestID string) *slog.Logger {
	return base.With(
		slog.String("request_id", requestID),
		slog.String("method", method),
		slog.String("path", path),
	)
}

// elapsedMS formats a duration in milliseconds for log attributes.
func elapsedMS(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}
