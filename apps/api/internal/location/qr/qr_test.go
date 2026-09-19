package qr

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/platform/apperr"
)

// fakePlaces implements the hospitalmap.Service surface the QR provider
// uses: GetPlace over a fixed table. Reception is routable; LAB-99 exists
// but has no entry node; anything else is not found.
type fakePlaces struct {
	gotPlaceID string
}

func (f *fakePlaces) GetPlace(_ context.Context, placeID string) (hospitalmap.Place, error) {
	f.gotPlaceID = placeID
	switch placeID {
	case "REG-01":
		node := "I-1301/node-reception"
		return hospitalmap.Place{ID: placeID, FloorID: "I-1301", Name: "Reception", EntryNodeID: &node}, nil
	case "LAB-99":
		return hospitalmap.Place{ID: placeID, FloorID: "I-1301", Name: "Old Lab"}, nil
	default:
		return hospitalmap.Place{}, hospitalmap.ErrPlaceNotFound
	}
}

func (f *fakePlaces) ListPlaces(context.Context) ([]hospitalmap.Place, error) { return nil, nil }

// The scan happy path: a location URL (full or bare path, with or without
// query) resolves through the place to its entry node.
func TestResolveURLForms(t *testing.T) {
	places := &fakePlaces{}
	provider := New(places)

	tests := []struct {
		name    string
		raw     string
		wantRef string
	}{
		{"full url", "https://carepath.example/location/REG-01", "REG-01"},
		{"bare path", "location/REG-01", "REG-01"},
		{"leading slash", "/location/REG-01", "REG-01"},
		{"query suffix", "https://carepath.example/location/REG-01?utm=poster", "REG-01"},
		{"trailing slash + whitespace", "  https://carepath.example/location/REG-01/ ", "REG-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs, err := provider.Resolve(context.Background(), tt.raw)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tt.raw, err)
			}
			if obs.NodeID != "I-1301/node-reception" {
				t.Fatalf("obs.NodeID = %q, want the REG-01 entry node", obs.NodeID)
			}
			if places.gotPlaceID != tt.wantRef {
				t.Fatalf("looked up place %q, want %q", places.gotPlaceID, tt.wantRef)
			}
		})
	}
}

// AC #1: whatever the provider needs, it must come from the payload alone —
// the observation carries no patient data, and the payload none either.
func TestResolveCarriesNoPatientData(t *testing.T) {
	provider := New(&fakePlaces{})

	obs, err := provider.Resolve(context.Background(), "https://carepath.example/location/REG-01")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if obs.VisitID != "" {
		t.Fatalf("obs.VisitID = %q, want empty — visit binding is the service's job", obs.VisitID)
	}
	if obs.NodeID != "I-1301/node-reception" {
		t.Fatalf("obs.NodeID = %q, want the entry node", obs.NodeID)
	}
}

// AC #4: payloads that do not resolve to a known location fail as invalid
// fixes, not as internal errors.
func TestResolveInvalidFixes(t *testing.T) {
	provider := New(&fakePlaces{})

	tests := []struct {
		name string
		raw  string
	}{
		{"empty", "   "},
		{"no location marker", "https://example.com/some/other/REG-01"},
		{"empty ref", "https://carepath.example/location/"},
		{"multi-segment ref", "https://carepath.example/location/foo/bar"},
		{"unknown place (stale QR)", "https://carepath.example/location/ROOM-404"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.Resolve(context.Background(), tt.raw)
			if !errors.Is(err, location.ErrInvalidFix) {
				t.Fatalf("Resolve(%q) error = %v, want ErrInvalidFix", tt.raw, err)
			}
			if apperr.KindOf(err) != apperr.KindInvalid {
				t.Fatalf("kind = %v, want Invalid", apperr.KindOf(err))
			}
		})
	}
}

func TestResolvePlaceWithoutEntryNode(t *testing.T) {
	provider := New(&fakePlaces{})

	// Known place that is not routable yet — handled, not a crash.
	if _, err := provider.Resolve(context.Background(), "location/LAB-99"); err != location.ErrInvalidFix {
		t.Fatalf("error = %v, want ErrInvalidFix for a place without an entry node", err)
	}
}

func TestSource(t *testing.T) {
	if got := New(&fakePlaces{}).Source(); got != location.SourceQR {
		t.Fatalf("Source() = %q, want QR", got)
	}
}
