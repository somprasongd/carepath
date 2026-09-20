package mockhis

import (
	"bytes"
	_ "embed"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/skip2/go-qrcode"

	"carepath/apps/mock-his/internal/platform/logger"
)

//go:embed console.html
var consoleHTML []byte

// New builds the Mock HIS HTTP app implementing the contract in
// packages/contracts/openapi/mock-his.yaml, plus the demo-driver surface
// (/api/v1/demo/* and /console) that exists only on the mock — a real HIS
// has its own operator tooling (see docs/integration/mock-his.md). The base
// logger feeds the request-logging middleware; state changes log through the
// request-scoped logger it stores in ctx.
func New(log *slog.Logger, patientAppBaseURL string) *fiber.App {
	store := NewStore()
	app := fiber.New()
	app.Use(logger.Middleware(log))

	// The console's "open patient view" link needs the patient web app's
	// origin to build an absolute URL when mock-his and web aren't
	// same-origin (local dev); inject it once at startup rather than
	// templating the page on every request.
	consolePage := consoleHTML
	if patientAppBaseURL != "" {
		script := []byte(`<script>window.PATIENT_APP_BASE_URL=` + strconv.Quote(patientAppBaseURL) + `;</script></head>`)
		consolePage = bytes.Replace(consoleHTML, []byte("</head>"), script, 1)
	}

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "mock-his"})
	})

	// Console: single embedded page, same origin as the API.
	app.Get("/", func(c fiber.Ctx) error {
		return c.Redirect().Status(http.StatusFound).To("/console")
	})
	app.Get("/console", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.Send(consolePage)
	})

	app.Get("/api/v1/visits/:visitId", func(c fiber.Ctx) error {
		visit, ok := store.GetVisit(c.Params("visitId"))
		if !ok {
			return errResponse(c, http.StatusNotFound, "visit not found")
		}
		return c.JSON(visit)
	})

	app.Get("/api/v1/events", func(c fiber.Ctx) error {
		limit := 50
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				return errResponse(c, http.StatusBadRequest, "invalid limit")
			}
			limit = parsed
		}
		events, next := store.Events(c.Query("after"), limit)
		return c.JSON(fiber.Map{"events": events, "nextAfter": next})
	})

	// Demo-driver API (mock-only, deliberately outside the canonical
	// contract): what the console uses to stage a demo. Every state change
	// is announced as a canonical event, so CarePath still learns everything
	// through the feed.
	app.Get("/api/v1/demo/visits", func(c fiber.Ctx) error {
		return c.JSON(store.ListVisits())
	})

	app.Post("/api/v1/demo/visits", func(c fiber.Ctx) error {
		var body struct {
			PatientRef  string `json:"patientRef"`
			PatientName string `json:"patientName"`
			VisitType   string `json:"visitType"`
			Clinics     []struct {
				ClinicCode string `json:"clinicCode"`
				ClinicName string `json:"clinicName"`
			} `json:"clinics"`
			Orders []struct {
				OrderType       string `json:"orderType"`
				OrderName       string `json:"orderName"`
				OrderedByClinic string `json:"orderedByClinic"`
			} `json:"orders"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return errResponse(c, http.StatusBadRequest, "invalid request body")
		}
		clinics := make([]OpenVisitClinic, len(body.Clinics))
		for i, cl := range body.Clinics {
			clinics[i] = OpenVisitClinic{Code: cl.ClinicCode, Name: cl.ClinicName}
		}
		orders := make([]OpenVisitOrder, len(body.Orders))
		for i, o := range body.Orders {
			orders[i] = OpenVisitOrder{OrderType: o.OrderType, OrderName: o.OrderName, OrderedByClinic: o.OrderedByClinic}
		}
		visit, verr := store.OpenVisit(c.Context(), body.PatientRef, body.PatientName, body.VisitType, clinics, orders)
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(visit)
	})

	app.Post("/api/v1/demo/visits/:visitId/clinics", func(c fiber.Ctx) error {
		var body struct {
			ClinicCode string `json:"clinicCode"`
			ClinicName string `json:"clinicName"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return errResponse(c, http.StatusBadRequest, "invalid request body")
		}
		visit, verr := store.AddClinic(c.Context(), c.Params("visitId"), body.ClinicCode, body.ClinicName)
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(visit)
	})

	app.Post("/api/v1/demo/visits/:visitId/orders", func(c fiber.Ctx) error {
		var body struct {
			OrderType       string `json:"orderType"`
			OrderName       string `json:"orderName"`
			OrderedByClinic string `json:"orderedByClinic"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return errResponse(c, http.StatusBadRequest, "invalid request body")
		}
		order, verr := store.PlaceOrder(c.Context(), c.Params("visitId"), body.OrderType, body.OrderName, body.OrderedByClinic)
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(order)
	})

	app.Post("/api/v1/demo/orders/:orderRef/performed", func(c fiber.Ctx) error {
		order, verr := store.MarkPerformed(c.Context(), c.Params("orderRef"))
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(order)
	})

	app.Post("/api/v1/demo/orders/:orderRef/resulted", func(c fiber.Ctx) error {
		order, verr := store.MarkResulted(c.Context(), c.Params("orderRef"))
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(order)
	})

	app.Post("/api/v1/demo/orders/:orderRef/cancel", func(c fiber.Ctx) error {
		order, verr := store.CancelOrder(c.Context(), c.Params("orderRef"))
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(order)
	})

	app.Post("/api/v1/demo/visits/:visitId/clinics/:clinicCode/complete-encounter", func(c fiber.Ctx) error {
		visit, verr := store.CompleteEncounter(c.Context(), c.Params("visitId"), c.Params("clinicCode"))
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(visit)
	})

	app.Post("/api/v1/demo/visits/:visitId/complete", func(c fiber.Ctx) error {
		visit, verr := store.CompleteVisit(c.Context(), c.Params("visitId"))
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(visit)
	})

	// Visit cancellation — demo story: the patient cancels the appointment,
	// so every open order and the visit itself are cancelled and announced
	// as canonical events.
	app.Post("/api/v1/demo/visits/:visitId/cancel", func(c fiber.Ctx) error {
		visit, verr := store.CancelVisit(c.Context(), c.Params("visitId"))
		if verr != nil {
			return storeErrResponse(c, verr)
		}
		return c.JSON(visit)
	})

	// QR for the console detail panel — demo stand-in for the real printed
	// navigation slip's QR: encodes the patient-view URL for this visit, so
	// scanning it (or clicking the console link) opens the journey directly.
	app.Get("/api/v1/demo/visits/:visitId/qrcode.png", func(c fiber.Ctx) error {
		visitID := c.Params("visitId")
		if _, ok := store.GetVisit(visitID); !ok {
			return errResponse(c, http.StatusNotFound, "visit not found")
		}
		target := patientAppBaseURL + "/patient/journey?visit=" + url.QueryEscape(visitID)
		png, err := qrcode.Encode(target, qrcode.Medium, 256)
		if err != nil {
			return errResponse(c, http.StatusInternalServerError, "qr encode failed")
		}
		c.Set(fiber.HeaderContentType, "image/png")
		return c.Send(png)
	})

	return app
}

func errResponse(c fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}

func storeErrResponse(c fiber.Ctx, err *Error) error {
	switch err.Kind {
	case ErrNotFound:
		return errResponse(c, http.StatusNotFound, err.Msg)
	case ErrConflict:
		return errResponse(c, http.StatusConflict, err.Msg)
	default:
		return errResponse(c, http.StatusBadRequest, err.Msg)
	}
}
