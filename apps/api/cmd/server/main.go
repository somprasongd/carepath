// Command server is the CarePath API composition root: it wires the postgres
// pool, the HIS client, and each module's adapters/services/handlers together,
// then serves HTTP. Business logic lives in internal/*, not here.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"
	// The analytics timezone (#86) resolves through the IANA database,
	// which the minimal container image does not ship — embed it so
	// ANALYTICS_TIMEZONE works in every deployment shape.
	_ "time/tzdata"

	"github.com/gofiber/fiber/v3"
	"github.com/swaggo/swag"

	_ "carepath/apps/api/docs"
	"carepath/apps/api/internal/analytics"
	analyticspostgres "carepath/apps/api/internal/analytics/postgres"
	"carepath/apps/api/internal/auth"
	authpostgres "carepath/apps/api/internal/auth/postgres"
	"carepath/apps/api/internal/his/httpclient"
	"carepath/apps/api/internal/his/ingest"
	ingestpostgres "carepath/apps/api/internal/his/ingest/postgres"
	"carepath/apps/api/internal/hospitalmap"
	hospitalmappostgres "carepath/apps/api/internal/hospitalmap/postgres"
	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/identity/line"
	"carepath/apps/api/internal/journey"
	journeypostgres "carepath/apps/api/internal/journey/postgres"
	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/location/manual"
	locationpostgres "carepath/apps/api/internal/location/postgres"
	"carepath/apps/api/internal/location/qr"
	"carepath/apps/api/internal/location/zigbee"
	"carepath/apps/api/internal/navigation"
	navigationpostgres "carepath/apps/api/internal/navigation/postgres"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/platform/logger"
	"carepath/apps/api/internal/servicepoint"
	servicepointpostgres "carepath/apps/api/internal/servicepoint/postgres"
	"carepath/apps/api/internal/session"
	sessionpostgres "carepath/apps/api/internal/session/postgres"
	"carepath/apps/api/internal/share"
	sharepostgres "carepath/apps/api/internal/share/postgres"
)

// @title			CarePath API
// @version		1.0.0
// @description	Patient journey and indoor navigation API above the HIS.
// @description	Serves the CarePath-derived journey plan (ADR-0009) with
// @description	every actionable step resolved to its service point.
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
	databaseURL := resolveDatabaseURL()
	database, err := db.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer database.Close()

	hisClient := httpclient.New(envOrDefault("HIS_BASE_URL", "http://localhost:8090"), nil)
	hospitalMap := hospitalmap.NewService(hospitalmappostgres.New(database))
	servicePoints := servicepoint.NewService(servicepointpostgres.New(database), hospitalMap)

	// Current location (#32/#33): scanned QR fixes, manual picks, and the
	// demo Zigbee simulator resolve through their provider onto a canonical
	// navigation node and become the visit's routing start point. The same
	// graph serves the route API (#28), which resolves the destination
	// service point through the servicepoint module.
	navigationGraph := navigation.NewService(navigationpostgres.New(database), servicePoints)
	locations, err := location.NewService(locationpostgres.New(database), navigationGraph,
		qr.New(hospitalMap, navigationGraph), manual.New(), zigbee.New(navigationGraph))
	if err != nil {
		return err
	}

	// Inbound HIS boundary (#21): poll the canonical event feed and keep the
	// journey projection in sync with the system of record. The queue stats
	// (#101) share the analytics timezone — it is the hospital's timezone,
	// not an analytics-specific knob.
	analyticsTZ := envOrDefault("ANALYTICS_TIMEZONE", "Asia/Bangkok")
	journeys := journey.NewService(hisClient, servicePoints, navigationGraph, locations, journeypostgres.New(database), database, analyticsTZ)
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
	// Visit ownership (#96): the claim is the bridge from a verified identity
	// to the visits it may read. Patient journey surfaces take this guard, so
	// a session can only reach visits it claimed by naming the VN.
	claims := sessionpostgres.NewClaimRepo(database)

	// Staff/admin auth (ADR-0010): argon2id passwords, a stateless 15-minute
	// JWT access token, and a rotating single-use refresh token. JWT_SECRET
	// has no default on purpose — an unset secret mints a random key at boot
	// (tokens then die with the process) and says so, instead of silently
	// sharing one hardcoded key with every deployment.
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = make([]byte, 32)
		if _, err := rand.Read(jwtSecret); err != nil {
			return fmt.Errorf("generate boot-time JWT secret: %w", err)
		}
		log.Warn("JWT_SECRET not set; using a random boot-time key — tokens will not survive a restart")
	}
	accessTokenTTL := envDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	refreshTokenTTL := envDuration("REFRESH_TOKEN_TTL", 168*time.Hour)
	authService := auth.NewService(authpostgres.New(database), database,
		auth.NewTokenIssuer(jwtSecret, accessTokenTTL), refreshTokenTTL)

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

	session.NewHandler(sessions, claims).Register(app.Group("/api/v1"))
	authHandler := auth.NewHandler(authService)
	authHandler.Register(app.Group("/api/v1"))
	// /auth/me sits behind the same guard as the staff surfaces: any
	// authenticated staff user, no particular role. The guard is per-route,
	// never group-level — fiber group middleware on /api/v1 would seal the
	// patient routes too.
	authHandler.RegisterMe(app.Group("/api/v1"), auth.RequireRole(authService))
	// The staff commands and the monitor read require STAFF or ADMIN; the
	// patient journey read requires the patient session and a claim on the
	// visit (#96, the ADR-0010 leftover) — a blanket guard would still break
	// the patient screens, so both stay per-route.
	patientVisitGuard := session.RequirePatientVisit(sessions, claims)
	staffGuard := auth.RequireRole(authService, auth.RoleStaff, auth.RoleAdmin)
	journey.NewHandler(journeys, servicePoints).Register(app.Group("/api/v1"),
		staffGuard,
		patientVisitGuard)
	// Executive analytics (#86): read-only aggregates over the timeline the
	// journey module writes (#85) — journey เขียน · analytics อ่าน. STAFF
	// and ADMIN keep their existing reach; EXECUTIVE logins reach this
	// surface and nothing else (they stay 403 on every /staff route above).
	analyticsService, err := analytics.NewService(analyticspostgres.New(database), database,
		analyticsTZ)
	if err != nil {
		return err
	}
	analytics.NewHandler(analyticsService).Register(app.Group("/api/v1"),
		auth.RequireRole(authService, auth.RoleStaff, auth.RoleAdmin, auth.RoleExecutive))
	// Relative share links (ADR-0011): the third credential kind. Creating
	// and revoking a link takes the patient visit guard (#96) — the session
	// must own the visit it shares; the shared read accepts the share token
	// itself and nothing else — cross-kind token use fails in every direction
	// by construction.
	shareTTL := envDuration("SHARE_LINK_TTL", 4*time.Hour)
	shares := share.NewService(sharepostgres.New(database), journeys, database, shareTTL)
	share.NewHandler(shares).Register(app.Group("/api/v1"), patientVisitGuard)
	// The public service-point reads stay unguarded; the assigned-points
	// read (/staff/my/service-points) is the queue console's picker and
	// shares the staff guard with the journey staff routes.
	servicepoint.NewHandler(servicePoints).Register(app.Group("/api/v1"), staffGuard)
	locationHandler := location.NewHandler(locations)
	locationHandler.Register(app.Group("/api/v1"), patientVisitGuard)
	locationHandler.RegisterDemo(app.Group("/api/v1"))
	navigation.NewHandler(navigationGraph).Register(app.Group("/api/v1"))

	return app.Listen(":" + envOrDefault("PORT", "8080"))
}

// resolveDatabaseURL builds the connection string's shape (scheme, host,
// port, path) from DATABASE_URL if set, otherwise from POSTGRES_HOST/PORT/DB
// (matching docker-compose's own defaults). Credentials always come from
// POSTGRES_USER/POSTGRES_PASSWORD and are layered on top, so DATABASE_URL
// never needs to carry secrets and POSTGRES_USER is never silently ignored.
func resolveDatabaseURL() string {
	raw := envOrDefault("DATABASE_URL", fmt.Sprintf("postgres://%s:%s/%s?sslmode=disable",
		envOrDefault("POSTGRES_HOST", "localhost"),
		envOrDefault("POSTGRES_PORT", "5432"),
		envOrDefault("POSTGRES_DB", "carepath"),
	))
	dsn, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
			dsn.User = url.UserPassword(user, password)
		} else {
			dsn.User = url.User(user)
		}
	}
	return dsn.String()
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
