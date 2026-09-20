// Package qr implements the ADR-0004 QR provider, the MVP baseline source.
// A QR code at a hospital point carries a stable map reference — a URL like
// https://carepath.example/location/REG-01 — and nothing else: no patient
// data. Two reference forms resolve: a place reference (the path segment
// after "/location/", the entry point of a service point's place) and a
// bare navigation-node reference ("node/<floorId>/<localId>", e.g. a lift or
// an entrance — points patients stand at that are not service places).
// Either way, resolving ends at a canonical navigation node.
package qr

import (
	"context"
	"strings"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
)

// Provider resolves scanned CarePath location QR payloads.
type Provider struct {
	places hospitalmap.Service
	nodes  navigation.Service
}

// New takes the hospital map place references resolve against and the
// navigation graph node references resolve against.
func New(places hospitalmap.Service, nodes navigation.Service) Provider {
	return Provider{places: places, nodes: nodes}
}

func (p Provider) Source() location.Source { return location.SourceQR }

func (p Provider) Resolve(ctx context.Context, raw string) (location.Observation, error) {
	if nodeID, ok := parseNodeRef(raw); ok {
		return p.resolveNode(ctx, nodeID)
	}
	placeID, err := parsePlaceRef(raw)
	if err != nil {
		return location.Observation{}, err
	}
	place, err := p.places.GetPlace(ctx, placeID)
	if err != nil {
		// A QR referencing a place that is gone (stale sticker after
		// re-modeling) or misscanned is an invalid fix, not a missing
		// resource: the client's input is what is wrong.
		if apperr.KindOf(err) == apperr.KindNotFound {
			return location.Observation{}, location.ErrInvalidFix
		}
		return location.Observation{}, apperr.Wrapf(apperr.KindInternal, err, "qr: look up place %q", placeID)
	}
	if place.EntryNodeID == nil || *place.EntryNodeID == "" {
		return location.Observation{}, location.ErrInvalidFix
	}
	return location.Observation{NodeID: *place.EntryNodeID}, nil
}

// resolveNode: the same bargain as a place reference, against the graph —
// an unknown node id is the sticker's (or the scan's) mistake.
func (p Provider) resolveNode(ctx context.Context, nodeID string) (location.Observation, error) {
	if _, err := p.nodes.GetNode(ctx, nodeID); err != nil {
		if apperr.KindOf(err) == apperr.KindNotFound {
			return location.Observation{}, location.ErrInvalidFix
		}
		return location.Observation{}, apperr.Wrapf(apperr.KindInternal, err, "qr: look up node %q", nodeID)
	}
	return location.Observation{NodeID: nodeID}, nil
}

// parseNodeRef extracts a bare navigation-node reference: everything after
// the "node/" marker up to any query or fragment. Node ids are globally
// unique as "<floorId>/<localId>", so the remainder may itself contain a
// slash; an empty remainder is not a reference.
func parseNodeRef(raw string) (string, bool) {
	payload := strings.TrimSpace(raw)
	if !strings.HasPrefix(payload, "node/") {
		return "", false
	}
	ref := payload[len("node/"):]
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	ref = strings.Trim(ref, "/")
	if ref == "" {
		return "", false
	}
	return ref, true
}

// parsePlaceRef extracts the place identifier from a scanned payload: the
// path segment after "/location/". A full URL, a bare path, and a payload
// with query or fragment all reduce to the same place id; anything without
// the marker (or with an empty/extra-segment ref) is not a CarePath
// location QR.
func parsePlaceRef(raw string) (string, error) {
	payload := strings.TrimSpace(raw)
	var ref string
	switch {
	case strings.Contains(payload, "/location/"):
		ref = payload[strings.Index(payload, "/location/")+len("/location/"):]
	case strings.HasPrefix(payload, "location/"):
		ref = payload[len("location/"):]
	default:
		return "", location.ErrInvalidFix
	}
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	ref = strings.Trim(ref, "/")
	if ref == "" || strings.Contains(ref, "/") {
		return "", location.ErrInvalidFix
	}
	return ref, nil
}
