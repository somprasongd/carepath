package floorplan

import (
	"context"
	"strings"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

// Service is the only entry point other modules may call. Its methods take a
// plain ctx: when the caller opened a transaction, repository calls made here
// join it through that ctx.
type Service interface {
	// SVG returns the stored plan for a floor and digest.
	SVG(ctx context.Context, floorID, sha256 string) (string, error)
	// Digests maps plan ids to their digests.
	Digests(ctx context.Context, planIDs []string) (map[string]string, error)
	// Upload normalizes an uploaded plan, stores it, and makes it the
	// floor's active one.
	Upload(ctx context.Context, floorID string, src []byte, actor string) (Stored, error)
	// Activate points the floor at a plan it already has — the rollback.
	Activate(ctx context.Context, floorID, planID string) (Stored, error)
	// ListPlans returns a floor's plans, newest first.
	ListPlans(ctx context.Context, floorID string) ([]Stored, error)
	// Raw returns a plan's bytes as uploaded, for an admin to download back.
	Raw(ctx context.Context, floorID, planID string) (string, error)
	// Health re-checks the floor's active plan against what the map model
	// currently believes, so the result reflects a place or node added or
	// removed since the plan was uploaded. Stored.Warnings, by contrast, is
	// a snapshot frozen at upload time — the historical record of what that
	// upload was accepted with, not the plan's current standing. Returns
	// nil when the floor has no active plan yet.
	Health(ctx context.Context, floorID string) ([]Warning, error)
}

type service struct {
	repo   Repo
	floors hospitalmap.Service
	graph  navigation.Service
	tx     db.Transactor
}

func NewService(repo Repo, floors hospitalmap.Service, graph navigation.Service, tx db.Transactor) Service {
	return &service{repo: repo, floors: floors, graph: graph, tx: tx}
}

func (s *service) SVG(ctx context.Context, floorID, sha256 string) (string, error) {
	return s.repo.SVG(ctx, floorID, sha256)
}

// Digests short-circuits an empty ask rather than sending an empty ANY() to
// Postgres — a floor with no plan yet is the normal state before the first
// upload, not an error.
func (s *service) Digests(ctx context.Context, planIDs []string) (map[string]string, error) {
	if len(planIDs) == 0 {
		return map[string]string{}, nil
	}
	return s.repo.Digests(ctx, planIDs)
}

func (s *service) ListPlans(ctx context.Context, floorID string) ([]Stored, error) {
	return s.repo.ListByFloor(ctx, floorID)
}

func (s *service) Raw(ctx context.Context, floorID, planID string) (string, error) {
	return s.repo.Raw(ctx, floorID, planID)
}

// Upload is the whole point of ADR-0015: bytes an admin sends become a
// trusted artifact, or they are refused.
//
// Storing the plan and pointing the floor at it happen in one transaction,
// which is the reason the artifact lives in Postgres rather than an object
// store — there is no half state where the bytes exist but no floor claims
// them, or a floor points at a plan that was never written. Resolving the
// floor, gathering what the map model expects, and normalizing the upload
// all happen before that transaction opens: none of them writes anything,
// so none of them needs to hold a write transaction open while it runs —
// this pool defaults to READ COMMITTED, so moving them out front changes
// nothing about what a concurrent write can do.
func (s *service) Upload(ctx context.Context, floorID string, src []byte, actor string) (Stored, error) {
	floor, err := s.floors.GetFloor(ctx, floorID)
	if err != nil {
		return Stored{}, err
	}
	expect, err := s.expectationsFor(ctx, floor)
	if err != nil {
		return Stored{}, err
	}
	plan, err := Normalize(src, expect)
	if err != nil {
		return Stored{}, err
	}

	// The id is derived from the content, so re-uploading an unchanged file
	// is idempotent: it lands on the row already there and simply re-points
	// the floor at it.
	planID := PlanIDFor(floorID, plan.SHA256)
	row := Stored{
		PlanID:   planID,
		FloorID:  floorID,
		SHA256:   plan.SHA256,
		ViewBox:  plan.ViewBox,
		Warnings: plan.Warnings,
	}
	if actor != "" {
		row.CreatedBy = &actor
	}

	var stored Stored
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Insert(ctx, row, plan.SVG, string(src)); err != nil {
			return err
		}
		if err := s.floors.SetPlan(ctx, floorID, plan.ViewBox, planID); err != nil {
			return err
		}

		var err error
		stored, err = s.repo.Get(ctx, floorID, planID)
		return err
	})
	if err != nil {
		return Stored{}, err
	}
	return stored, nil
}

// Activate moves the floor's pointer to a plan it already holds. Plans are
// append-only, so this is the whole of "roll back to the previous drawing".
func (s *service) Activate(ctx context.Context, floorID, planID string) (Stored, error) {
	var stored Stored
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		plan, err := s.repo.Get(ctx, floorID, planID)
		if err != nil {
			return err
		}
		if err := s.floors.SetPlan(ctx, floorID, plan.ViewBox, planID); err != nil {
			return err
		}
		stored = plan
		return nil
	})
	if err != nil {
		return Stored{}, err
	}
	return stored, nil
}

// Health re-normalizes the floor's active plan against a freshly gathered
// Expect, which is the same check Upload runs but against the map model as
// it stands right now rather than as it stood at upload time. The plan
// itself cannot fail that re-check — it already passed the allowlist and
// its own viewBox/data-floor once — so this only ever recomputes Warnings.
func (s *service) Health(ctx context.Context, floorID string) ([]Warning, error) {
	floor, err := s.floors.GetFloor(ctx, floorID)
	if err != nil {
		return nil, err
	}
	if floor.ActivePlanID == nil {
		return []Warning{}, nil
	}
	active, err := s.repo.Get(ctx, floorID, *floor.ActivePlanID)
	if err != nil {
		return nil, err
	}
	svg, err := s.repo.SVG(ctx, floorID, active.SHA256)
	if err != nil {
		return nil, err
	}
	expect, err := s.expectationsFor(ctx, floor)
	if err != nil {
		return nil, err
	}
	plan, err := Normalize([]byte(svg), expect)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "floorplan: re-check active plan on floor %q", floorID)
	}
	return plan.Warnings, nil
}

// expectationsFor gathers what the map model already believes about this
// floor, so the upload is checked against it while the admin is still at the
// keyboard rather than discovered by a patient in a corridor.
func (s *service) expectationsFor(ctx context.Context, floor hospitalmap.Floor) (Expect, error) {
	expect := Expect{FloorID: floor.ID}
	if floor.ViewBox != nil {
		expect.ViewBox = *floor.ViewBox
	}

	places, err := s.floors.ListPlaces(ctx)
	if err != nil {
		return Expect{}, apperr.Wrapf(apperr.KindInternal, err, "floorplan: list places")
	}
	for _, place := range places {
		if place.FloorID == floor.ID {
			expect.PlaceIDs = append(expect.PlaceIDs, place.ID)
		}
	}

	nodes, err := s.graph.ListNodes(ctx)
	if err != nil {
		return Expect{}, apperr.Wrapf(apperr.KindInternal, err, "floorplan: list navigation nodes")
	}
	for _, node := range nodes {
		if node.FloorID != floor.ID {
			continue
		}
		// Graph ids are globally unique ("I-1301/node-lift") because the
		// authored ids are floor-local and node-lift exists on both floors.
		// An SVG carries the floor-local form, so compare in that form.
		expect.NodeIDs = append(expect.NodeIDs, strings.TrimPrefix(node.ID, floor.ID+"/"))
	}
	return expect, nil
}

// PlanIDFor keys a plan by its floor and its full content digest — not a
// prefix, so two distinct plans cannot collide onto the same id and have
// Insert's ON CONFLICT silently keep the wrong one. cmd/floorplanseed calls
// this same function, so a seeded plan and the same file uploaded through
// the API are the same row, not two.
func PlanIDFor(floorID, sha256 string) string {
	return "FP-" + strings.ReplaceAll(floorID, "-", "") + "-" + sha256
}
