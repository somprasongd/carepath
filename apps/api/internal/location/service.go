package location

import (
	"context"
	"fmt"
	"sort"
	"time"

	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
)

// Service is the only entry point other modules may call. Resolve turns a
// raw provider fix into a validated canonical observation; Report also
// records it as the visit's current location; Current reads that location
// back. Its methods take a plain ctx: when the caller opened a transaction,
// repository calls made here join it through that ctx.
type Service interface {
	// Sources lists the sources with a registered provider.
	Sources() []Source
	Resolve(ctx context.Context, source Source, raw string) (Observation, error)
	Report(ctx context.Context, visitID string, source Source, raw string) (Observation, error)
	Current(ctx context.Context, visitID string) (Observation, error)
}

type service struct {
	repo       Repo
	navigation navigation.Service
	providers  map[Source]Provider
}

// NewService registers one provider per source. A duplicate source is a
// wiring bug and fails construction instead of silently shadowing.
func NewService(repo Repo, nav navigation.Service, providers ...Provider) (Service, error) {
	s := &service{
		repo:       repo,
		navigation: nav,
		providers:  make(map[Source]Provider, len(providers)),
	}
	for _, p := range providers {
		src := p.Source()
		if _, exists := s.providers[src]; exists {
			return nil, apperr.New(apperr.KindInternal, fmt.Sprintf("duplicate location provider for source %q", src))
		}
		s.providers[src] = p
	}
	return s, nil
}

func (s *service) Sources() []Source {
	sources := make([]Source, 0, len(s.providers))
	for src := range s.providers {
		sources = append(sources, src)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i] < sources[j] })
	return sources
}

func (s *service) Resolve(ctx context.Context, source Source, raw string) (Observation, error) {
	provider, ok := s.providers[source]
	if !ok {
		return Observation{}, ErrUnknownSource
	}
	obs, err := provider.Resolve(ctx, raw)
	if err != nil {
		return Observation{}, err
	}
	obs.Source = source

	// Canonicalize against the navigation graph: the fix must name a real
	// node, and floor/zone come from the graph rather than provider input.
	node, err := s.navigation.GetNode(ctx, obs.NodeID)
	if err != nil {
		if apperr.KindOf(err) == apperr.KindNotFound {
			return Observation{}, ErrUnknownNode
		}
		return Observation{}, apperr.Wrapf(apperr.KindInternal, err, "location: look up node %q", obs.NodeID)
	}
	obs.FloorID = node.FloorID
	if obs.Zone == nil {
		obs.Zone = node.Zone
	}
	if obs.ObservedAt.IsZero() {
		obs.ObservedAt = time.Now()
	}
	if err := obs.Validate(); err != nil {
		return Observation{}, err
	}
	return obs, nil
}

func (s *service) Report(ctx context.Context, visitID string, source Source, raw string) (Observation, error) {
	if visitID == "" {
		return Observation{}, apperr.New(apperr.KindInvalid, "visitId is required to report a location")
	}
	obs, err := s.Resolve(ctx, source, raw)
	if err != nil {
		return Observation{}, err
	}
	obs.VisitID = visitID
	if err := s.repo.Record(ctx, obs); err != nil {
		return Observation{}, err
	}
	return obs, nil
}

func (s *service) Current(ctx context.Context, visitID string) (Observation, error) {
	return s.repo.Latest(ctx, visitID)
}
