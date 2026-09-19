// Package servicepoint owns the link between a care step and a physical
// place. Other modules must depend on its Service interface, never on its
// Repo — the Repo is this module's internal persistence contract.
package servicepoint

import (
	"context"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrNotFound is returned when no active service point matches the lookup.
var ErrNotFound = apperr.New(apperr.KindNotFound, "service point not found")

// ServicePoint links a service code from a care step to a Place in the
// hospital map (see docs/architecture/domain-model.md). Active is a
// persistence concern and is excluded from JSON responses.
type ServicePoint struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	PlaceID string `json:"placeId"`
	Active  bool   `json:"-"`
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; callers get a tx-bound Querier via the ctx passed to
// Service methods, so repository calls always join the ambient transaction.
type Repo interface {
	GetByCode(ctx context.Context, code string) (ServicePoint, error)
	List(ctx context.Context) ([]ServicePoint, error)
}
