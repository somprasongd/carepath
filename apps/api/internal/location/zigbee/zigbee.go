// Package zigbee implements the ADR-0004 Zigbee source as a simulator: the
// raw fix is a JSON message shaped like a positioning-service output (see
// docs/integration/location-zigbee.md) — floor, zone, optional confidence —
// and resolving means placing the tag inside the zone's representative node
// of the navigation graph. It exists so demos and tests can drive zone-level
// location updates without hardware (#33); a real Zigbee integration
// (Zigbee2MQTT → positioning service → CarePath) will be a separate adapter
// implementing the same location.Provider port, not a change to this
// package.
package zigbee

import (
	"context"
	"encoding/json"
	"strings"

	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
)

// Provider resolves simulated Zigbee positioning fixes against the
// navigation graph.
type Provider struct {
	graph navigation.Service
}

// New takes the navigation graph whose zones the fixes resolve in.
func New(graph navigation.Service) Provider {
	return Provider{graph: graph}
}

func (p Provider) Source() location.Source { return location.SourceZigbee }

// Fix is the raw fix format: a simulated positioning-service message. Floor
// and zone say where the tag is; confidence (0..1, optional) says how sure
// the positioning is — the metadata that separates a probabilistic Zigbee
// fix from an exact QR or manual one.
type Fix struct {
	FloorID    string   `json:"floorId"`
	Zone       string   `json:"zone"`
	Confidence *float64 `json:"confidence,omitempty"`
}

func (p Provider) Resolve(ctx context.Context, raw string) (location.Observation, error) {
	var fix Fix
	if err := json.Unmarshal([]byte(raw), &fix); err != nil {
		return location.Observation{}, location.ErrInvalidFix
	}
	fix.FloorID = strings.TrimSpace(fix.FloorID)
	fix.Zone = strings.TrimSpace(fix.Zone)
	if fix.FloorID == "" || fix.Zone == "" {
		return location.Observation{}, location.ErrInvalidFix
	}

	nodes, err := p.graph.ListNodes(ctx)
	if err != nil {
		return location.Observation{}, apperr.Wrapf(apperr.KindInternal, err, "zigbee: load navigation nodes")
	}
	node, ok := nodeInZone(nodes, fix.FloorID, fix.Zone)
	if !ok {
		// A zone the graph does not know (wrong floor, renamed area) is a
		// bad fix, not a missing resource — the same stance the QR provider
		// takes on a stale place.
		return location.Observation{}, location.ErrInvalidFix
	}
	return location.Observation{NodeID: node.ID, Zone: node.Zone, Confidence: fix.Confidence}, nil
}

// nodeInZone picks the zone's representative node on the floor: the first
// PLACE_ENTRY by id, else the first ENTRANCE, else the first remaining
// node — deterministic, so the same fix always resolves to the same place.
func nodeInZone(nodes []navigation.NavNode, floorID, zone string) (navigation.NavNode, bool) {
	var best navigation.NavNode
	bestRank := 99
	found := false
	for _, n := range nodes {
		if n.FloorID != floorID || n.Zone == nil || !strings.EqualFold(*n.Zone, zone) {
			continue
		}
		if rank := zoneRank(n); rank < bestRank || (rank == bestRank && n.ID < best.ID) {
			best, bestRank, found = n, rank, true
		}
	}
	return best, found
}

// zoneRank prefers a destination node of the zone, then an entrance, then
// anything else (corridors, transitions).
func zoneRank(n navigation.NavNode) int {
	switch n.NodeType {
	case "PLACE_ENTRY":
		return 0
	case "ENTRANCE":
		return 1
	default:
		return 2
	}
}
