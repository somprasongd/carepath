// Package manual implements the ADR-0004 manual-selection provider: the
// fallback/debug source where the raw fix is a canonical navigation node id
// picked by a human (staff selecting on a map, demo walk-throughs). It also
// serves as the reference implementation of the location.Provider port.
package manual

import (
	"context"
	"strings"

	"carepath/apps/api/internal/location"
)

// Provider resolves manually selected navigation node ids.
type Provider struct{}

// New returns the manual provider.
func New() Provider { return Provider{} }

func (p Provider) Source() location.Source { return location.SourceManual }

func (p Provider) Resolve(_ context.Context, raw string) (location.Observation, error) {
	nodeID := strings.TrimSpace(raw)
	if nodeID == "" {
		return location.Observation{}, location.ErrInvalidFix
	}
	return location.Observation{NodeID: nodeID}, nil
}
