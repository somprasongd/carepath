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

// Floor is one floor of a building, identified by the floor-plan id
// (e.g. I-1301). LevelOrder is the vertical display/sort order.
type Floor struct {
	ID         string `json:"id"`
	BuildingID string `json:"buildingId"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	LevelOrder int    `json:"levelOrder"`
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
}
