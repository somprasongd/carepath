// Package servicepoint owns the link between a care step and a physical
// place. Other modules must depend on its Service interface, never on its
// Repo — the Repo is this module's internal persistence contract.
package servicepoint

import (
	"context"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/platform/apperr"
)

// ErrNotFound is returned when no active service point matches the lookup.
var ErrNotFound = apperr.New(apperr.KindNotFound, "service point not found")

// ServicePoint links a service code from a care step to a Place in the
// hospital map (see docs/architecture/domain-model.md). Place is the
// resolved destination — nil when the place is not in the map (yet), the
// same explicit unmapped state as a nil ServicePoint on a journey step.
// Active is a persistence concern and is excluded from JSON responses.
type ServicePoint struct {
	ID      string             `json:"id"`
	Code    string             `json:"code"`
	Name    string             `json:"name"`
	PlaceID string             `json:"placeId"`
	Place   *hospitalmap.Place `json:"place,omitempty"`
	Active  bool               `json:"-"`
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; callers get a tx-bound Querier via the ctx passed to
// Service methods, so repository calls always join the ambient transaction.
type Repo interface {
	GetByCode(ctx context.Context, code string) (ServicePoint, error)
	List(ctx context.Context) ([]ServicePoint, error)
	// ListForUser returns the active service points assigned to the staff
	// user through user_service_point (#102), ordered by code.
	ListForUser(ctx context.Context, userID string) ([]ServicePoint, error)
	// IsAssigned reports whether the user is assigned to the service point.
	IsAssigned(ctx context.Context, userID, servicePointID string) (bool, error)
}
