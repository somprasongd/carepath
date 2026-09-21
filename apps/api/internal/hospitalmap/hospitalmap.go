// Package hospitalmap owns the spatial side of the hospital: buildings,
// floors, and the places service points resolve to (see
// docs/architecture/domain-model.md and ADR-0002 — spatial data stays
// separate from care flow). Other modules must depend on its Service
// interface, never on its Repo.
package hospitalmap

import (
	"context"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrPlaceNotFound is returned when no place matches the lookup.
var ErrPlaceNotFound = apperr.New(apperr.KindNotFound, "place not found")

// ErrFloorNotFound is returned when no floor matches the lookup.
var ErrFloorNotFound = apperr.New(apperr.KindNotFound, "floor not found")

// Floor is one floor of a building, identified by the floor-plan id
// (e.g. I-1301). LevelOrder is the vertical display/sort order.
type Floor struct {
	ID         string `json:"id"`
	BuildingID string `json:"buildingId"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	LevelOrder int    `json:"levelOrder"`
	// ViewBox is the floor's coordinate space ("0 0 1600 900"), the space
	// every Place X/Y and every nav node coordinate is expressed in. It is
	// set by the floor's first plan and holds every later one to it
	// (ADR-0015); nil on a floor that has no plan yet.
	ViewBox *string `json:"viewBox,omitempty"`
	// ActivePlanID names the floor_plan row currently served. Plans are
	// append-only, so switching or rolling one back moves this pointer
	// rather than editing anything. It is not part of a floor's public
	// shape: clients get the plan through the URL the floors listing
	// carries, never by plan id.
	ActivePlanID *string `json:"-"`
}

// Place is a physical place on a floor plan, identified by the stable SVG
// place id (e.g. REG-01) that also keys the navigation graphs in
// packages/floorplans. X/Y are floor-local SVG units of the place's entry
// node; EntryNodeID references that node in the navigation graph
// (nav_node, #26) as a floor-prefixed global id, e.g. I-1301/node-reception.
// Floor is resolved on read.
type Place struct {
	ID          string   `json:"id"`
	FloorID     string   `json:"floorId"`
	Name        string   `json:"name"`
	PlaceType   string   `json:"type"`
	X           *float64 `json:"x,omitempty"`
	Y           *float64 `json:"y,omitempty"`
	EntryNodeID *string  `json:"entryNodeId,omitempty"`
	Floor       *Floor   `json:"floor,omitempty"`
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; callers get a tx-bound Querier via the ctx passed to
// Service methods, so repository calls always join the ambient transaction.
type Repo interface {
	GetPlace(ctx context.Context, placeID string) (Place, error)
	ListPlaces(ctx context.Context) ([]Place, error)
	// ListFloors returns every floor in display order, with the plan
	// pointer the web app needs to fetch a drawing (#105).
	ListFloors(ctx context.Context) ([]Floor, error)
	GetFloor(ctx context.Context, floorID string) (Floor, error)
	// SetPlan points the floor at a plan, fixing its coordinate space at
	// the same time. The floor table is this module's, so the floorplan
	// module moves the pointer through here rather than writing it (#105).
	SetPlan(ctx context.Context, floorID, viewBox, planID string) error
}
