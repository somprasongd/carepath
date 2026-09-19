// Package mock provides a scripted location.Provider for tests — the mock
// provider of issue #30's acceptance criteria. It records the raw fixes it
// was asked to resolve and replays a canned observation or error, so the
// location service and its future consumers (#32 QR flow, #33 Zigbee) can
// test provider dispatch without real hardware. It is not safe for
// concurrent use; tests drive it from one goroutine.
package mock

import (
	"context"

	"carepath/apps/api/internal/location"
)

// Provider is a configurable location.Provider. New is the common case
// (every fix resolves to one node under one source); set Fix/Err directly
// for scripted variations. VisitID on Fix is ignored — the service owns it.
type Provider struct {
	// Src is the source this provider registers under.
	Src location.Source
	// Fix is the observation Resolve returns (Source and VisitID are
	// overwritten by the mock or the service).
	Fix location.Observation
	// Err, when set, is returned from Resolve instead of Fix.
	Err error
	// Raws records every raw fix Resolve received, in order.
	Raws []string
}

// New returns a provider resolving every raw fix to nodeID under source.
func New(source location.Source, nodeID string) *Provider {
	return &Provider{Src: source, Fix: location.Observation{NodeID: nodeID}}
}

func (p *Provider) Source() location.Source { return p.Src }

func (p *Provider) Resolve(_ context.Context, raw string) (location.Observation, error) {
	p.Raws = append(p.Raws, raw)
	if p.Err != nil {
		return location.Observation{}, p.Err
	}
	obs := p.Fix
	obs.VisitID = ""
	return obs, nil
}
