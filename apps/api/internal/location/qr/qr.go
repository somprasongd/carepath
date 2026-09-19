// Package qr implements the ADR-0004 QR provider, the MVP baseline source.
// A QR code at a hospital point carries a stable place reference — a URL
// like https://carepath.example/location/REG-01 (docs/integration/
// location-zigbee.md) — and nothing else: no patient data. Resolving means
// finding that place on the hospital map and returning its entry node as
// the canonical location.
package qr

import (
	"context"
	"strings"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/platform/apperr"
)

// Provider resolves scanned CarePath location QR payloads.
type Provider struct {
	places hospitalmap.Service
}

// New takes the hospital map the place references resolve against.
func New(places hospitalmap.Service) Provider {
	return Provider{places: places}
}

func (p Provider) Source() location.Source { return location.SourceQR }

func (p Provider) Resolve(ctx context.Context, raw string) (location.Observation, error) {
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
