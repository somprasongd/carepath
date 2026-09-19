package manual

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/location"
)

func TestSource(t *testing.T) {
	if got := New().Source(); got != location.SourceManual {
		t.Fatalf("Source() = %q, want MANUAL", got)
	}
}

func TestResolveTrimsInput(t *testing.T) {
	obs, err := New().Resolve(context.Background(), "  I-1301/node-reception \n")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if obs.NodeID != "I-1301/node-reception" {
		t.Fatalf("obs.NodeID = %q, want the trimmed node id", obs.NodeID)
	}
}

func TestResolveEmptyFix(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		if _, err := New().Resolve(context.Background(), raw); !errors.Is(err, location.ErrInvalidFix) {
			t.Fatalf("Resolve(%q) error = %v, want ErrInvalidFix", raw, err)
		}
	}
}
