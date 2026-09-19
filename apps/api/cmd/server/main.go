// Command server is the CarePath API composition root: it wires the postgres
// pool, the HIS client, and each module's adapters/services/handlers together,
// then serves HTTP. Business logic lives in internal/*, not here.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/swaggo/swag"

	_ "carepath/apps/api/docs"
	"carepath/apps/api/internal/his/httpclient"
	"carepath/apps/api/internal/his/ingest"
	ingestpostgres "carepath/apps/api/internal/his/ingest/postgres"
	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/identity/line"
	"carepath/apps/api/internal/journey"
	journeypostgres "carepath/apps/api/internal/journey/postgres"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/platform/logger"
	"carepath/apps/api/internal/servicepoint"
	servicepointpostgres "carepath/apps/api/internal/servicepoint/postgres"
	"carepath/apps/api/internal/session"
	sessionpostgres "carepath/apps/api/internal/session/postgres"
	"carepath/apps/api/internal/visit"
)

// @title			CarePath API
// @version		1.0.0
// @description	Patient journey and indoor navigation API above the HIS.
// @description	Serves the normalized visit view with the next actionable
// @description	step and its service point.
//
// @BasePath	/
func main() {
	log := logger.New()
	slog.SetDefault(log)
	if err := run(context.Background(), log); err != nil {
		log.Error("server exited", "error", err.Error())
		os.Exit(1)
	}
}

// HealthResponse reports service liveness.
type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"carepath-api"`
}

func run(ctx context.Context, log *slog.Logger) error {
	databaseURL := envOrDefault("DATABASE_URL", "postgres://carepath:carepath@localhost:5432/carepath?sslmode=disable")
	database, err := db.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	hisClient := httpclient.New(envOrDefault("HIS_BASE_URL", "http://localhost:8090"), nil)
	servicePoints := servicepoint.NewService(servicepointpostgres.New(database))
	visits := visit.NewService(hisClient, servicePoints, database)

	// Inbound HIS boundary (#21): poll the canonical event feed and keep the
	// journey projection in sync with the system of record.
	journeys := journey.NewService(hisClient, servicePoints, journeypostgres.New(database), database)
	poller := ingest.New(hisClient, journeys, ingestpostgres.New(database), log)
	interval := envDuration("HIS_INGEST_INTERVAL", 5*time.Second)
	go poller.Run(context.Background(), interval)
	log.Info("HIS ingest poller started", "interval", interval.String())

	// Identity/session (#15/#16): LINE_CHANNEL_ID is only required for real
	// LINE logins — a demo-only deployment can leave it unset, in which case
	// "line" source requests fail clearly (ErrLineAuthNotConfigured) instead
	// of the server refusing to boot.
	var lineVerifier identity.Verifier
	if channelID := os.Getenv("LINE_CHANNEL_ID"); channelID != "" {
		verifier, err := line.New(ctx, channelID, line.JWKSURL)
		if err != nil {
			return err
		}
		lineVerifier = verifier
	} else {
		log.Warn("LINE_CHANNEL_ID not set; source=line session requests will fail")
	}
	allowDemoAuth := envOrDefault("ALLOW_DEMO_AUTH", "false") == "true"
	sessionTTL := envDuration("SESSION_TTL", 24*time.Hour)
	sessions := session.NewService(sessionpostgres.New(database), lineVerifier, allowDemoAuth, sessionTTL)

	app := fiber.New()
	app.Use(logger.Middleware(log))
	app.Use(func(c fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})

	app.Get("/health", health)

	// The generated docs package registers the spec with swaggo/swag; served
	// raw so the Swagger UI below (and any tool) can consume it.
	app.Get("/api/openapi.json", func(c fiber.Ctx) error {
		doc, err := swag.ReadDoc()
		if err != nil {
			return err
		}
		return c.Type("json").SendString(doc)
	})
	app.Get("/swagger", func(c fiber.Ctx) error {
		return c.Type("html").SendString(swaggerUIPage)
	})

	visit.NewHandler(visits).Register(app.Group("/api/v1"))
	session.NewHandler(sessions).Register(app.Group("/api/v1"))
	journey.NewHandler(journeys).Register(app.Group("/api/v1"))

	return app.Listen(":" + envOrDefault("PORT", "8080"))
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log := slog.Default()
		log.Warn("invalid duration in env; using fallback", "key", key, "value", value, "fallback", fallback.String())
		return fallback
	}
	return parsed
}

// health godoc
//
//	@Summary		Health check
//	@Description	Liveness probe for the API.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func health(c fiber.Ctx) error {
	return c.JSON(HealthResponse{Status: "ok", Service: "carepath-api"})
}

const swaggerUIPage = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>CarePath API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => SwaggerUIBundle({ url: '/api/openapi.json', dom_id: '#swagger-ui' });
  </script>
</body>
</html>`
