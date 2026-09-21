package hospitalmap

import (
	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/platform/httpx"
)

// Handler exposes the hospital map's places. Floors are served by the
// floorplan module instead, because that response is the plan pointer
// document and has to carry a plan URL this module knows nothing about.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the hospital map routes under the given /api/v1 router.
func (h *Handler) Register(router fiber.Router) {
	router.Get("/places", h.listPlaces)
}

// listPlaces godoc
//
//	@Summary		List places
//	@Description	#105: every place in the hospital map with its floor resolved — the destinations a service point can be mapped to. Already reachable one at a time through a service point's resolved place; this is the whole set, for choosing among them. A place whose plan has not drawn it yet is still listed: it is selectable, just not routable.
//	@Tags			places
//	@Produce		json
//	@Success		200	{array}		hospitalmap.Place
//	@Failure		500	{object}	httpx.ErrorResponse	"internal server error"
//	@Router			/api/v1/places [get]
func (h *Handler) listPlaces(c fiber.Ctx) error {
	places, err := h.service.ListPlaces(c.Context())
	if err != nil {
		return httpx.Error(c, err)
	}
	return c.JSON(places)
}
