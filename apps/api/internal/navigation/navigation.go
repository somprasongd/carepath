// Package navigation owns the walkable graph of the hospital: nodes placed
// on floors and the directed, costed edges between them (see
// docs/architecture/technical-blueprint.md and ADR-0002 — wayfinding data
// stays separate from care flow). The graph is hand-authored in
// packages/floorplans/graphs and seeded from there; it is never computed
// from SVG geometry. Other modules must depend on its Service interface,
// never on its Repo — the Repo is this module's persistence contract.
package navigation

import (
	"context"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrNodeNotFound is returned when no node matches the lookup.
var ErrNodeNotFound = apperr.New(apperr.KindNotFound, "navigation node not found")

// NavNode is one walkable point of the hospital: a floor, a coordinate in
// that floor's SVG space, and what kind of point it is. ID is globally
// unique as "<floorId>/<localId>" (the JSON/SVG node ids are floor-local —
// node-lift exists on both floors). A PLACE_ENTRY node is the destination a
// Place resolves to through place.entry_node_id; Zone carries the
// positioning area from the graph JSON for the future location providers.
type NavNode struct {
	ID       string  `json:"id"`
	FloorID  string  `json:"floorId"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	NodeType string  `json:"nodeType"` // ENTRANCE, PLACE_ENTRY, CORRIDOR, ELEVATOR, STAIRS
	Zone     *string `json:"zone,omitempty"`
}

// NavEdge is one directed walk between neighbouring nodes with its cost.
// Two-way connections are two rows (A→B and B→A); a cross-floor transition
// is just an edge whose endpoints sit on different floors. Distance is
// authored schematic SVG units (not metres — the plans are not to scale);
// Accessible marks wheelchair-friendly paths (stairs are not).
type NavEdge struct {
	ID         string  `json:"id"`
	FromNodeID string  `json:"fromNodeId"`
	ToNodeID   string  `json:"toNodeId"`
	EdgeType   string  `json:"edgeType"` // CORRIDOR, ELEVATOR, STAIRS
	Distance   float64 `json:"distance"`
	Accessible bool    `json:"accessible"`
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; callers get a tx-bound Querier via the ctx passed to
// Service methods, so repository calls always join the ambient transaction.
type Repo interface {
	GetNode(ctx context.Context, nodeID string) (NavNode, error)
	ListNodes(ctx context.Context) ([]NavNode, error)
	ListEdges(ctx context.Context) ([]NavEdge, error)
}
