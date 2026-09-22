package navigation

import (
	"sort"

	"carepath/apps/api/internal/hospitalmap"
)

// AmenityNearby is one ranked amenity: the place itself — the same shape
// /api/v1/places serves, floor included — and its walking distance from the
// search origin in authored SVG units, the unit NavigationRoute.TotalDistance
// already sums (#109 / FR-26).
type AmenityNearby struct {
	Place    hospitalmap.Place `json:"place"`
	Distance float64           `json:"distance"`
}

// AmenitySearch is the answer of GET /navigation/amenities: amenity-typed
// places reachable from the origin, nearest first. Amenities is never null —
// "no amenity reachable" is an empty list, not an error.
type AmenitySearch struct {
	From      string          `json:"from"`
	Amenities []AmenityNearby `json:"amenities"`
}

// rankAmenities pairs amenity places with their settled distances and sorts
// nearest-first. Places without an entry node, or unreachable from the
// origin, are dropped — missing data is "no distance", not an error, the
// same rule DistancesToServicePoints applies. Ties fall back to place-id
// order so the ranking is stable across calls.
func rankAmenities(places []hospitalmap.Place, distances map[string]float64) []AmenityNearby {
	ranked := make([]AmenityNearby, 0, len(places))
	for _, place := range places {
		if place.EntryNodeID == nil {
			continue
		}
		distance, ok := distances[*place.EntryNodeID]
		if !ok {
			continue
		}
		ranked = append(ranked, AmenityNearby{Place: place, Distance: distance})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Distance != ranked[j].Distance {
			return ranked[i].Distance < ranked[j].Distance
		}
		return ranked[i].Place.ID < ranked[j].Place.ID
	})
	return ranked
}
