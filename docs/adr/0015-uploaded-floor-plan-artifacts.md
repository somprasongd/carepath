# ADR-0015: Floor Plans Are Uploaded, Normalized, and Served as Immutable Artifacts

- Status: Accepted
- Date: 2026-09-20
- Supersedes: [ADR-0003](0003-svg-floor-plan.md) (SVG stays; "a file in the repo" does not)
- Relates to: FR-11 (#105), [ADR-0002](0002-separate-care-and-navigation-graphs.md) (the graph stays a separate model), [ADR-0012](0012-client-owned-display-text.md) §4 (FR-11 is the trigger it names), [ADR-0010](0010-staff-auth-jwt-argon2.md) (ADMIN guard)

## Context

FR-11 says staff must eventually configure buildings, floors, places and
service points, and adds that a full CRUD UI is not required for the
hackathon. Reading the code against that, the blocking constraint is not
missing CRUD endpoints — it is where the floor-plan assets live:

- `apps/web` imports both the plans and the navigation graphs **at build
  time** (`features/floorplan/plans.ts`, `features/floorplan/graphs.ts`,
  `?raw` imports out of `packages/floorplans`), and hardcodes the floor list
  in `FLOOR_CODES`.
- So an admin who adds a place through any API still gets a patient app whose
  bundled SVG has no `data-place-id` for it. `floorPlanHasPlace()` returns
  false and the place degrades to "not routable" — correctly, but
  permanently.
- `wayfindingAnchors()` derives QR sticker payloads from the bundled graph
  JSON. Those stickers get printed and stuck to walls: a build-time copy that
  drifts from the database sends patients to nodes that no longer exist.

Admin map configuration is therefore an asset-pipeline decision before it is
a CRUD decision, and ADR-0003's "SVG floor plans for the MVP" was written
when those files could only come from a pull request.

The risk this opens is specific. `FloorPlanMap` injects the plan into the DOM
(`host.innerHTML`, `dangerouslySetInnerHTML`) because the destination
highlight and the drawn route need real elements to attach to; its own
comment states the precondition — *"The plan is a repo-controlled static
asset, not user input"*. Uploading removes that precondition on a page that
holds a patient session token.

## Decision

### 1. An upload is a compile step, not a storage step

Uploaded bytes are never stored or served as received. `internal/floorplan`
parses them, holds them to a closed allowlist, and **re-serializes the result
from the parsed tree**. What a patient's browser inlines is output CarePath
wrote, not input someone supplied.

Re-serializing rather than stripping is the load-bearing part: a sanitizer
that edits the source text has to out-guess every parser quirk that turns
harmless-looking bytes into markup, while a writer that can only emit
allowlisted names cannot emit anything else regardless of what the input did.

Consequently the page keeps inlining the plan. Rendering it through `<img>`
would be safe too, but it would take the route overlay and the destination
pin with it — the product's core, traded away for a property normalization
already provides.

Rejections are rejections, not silent repairs: an admin is at the keyboard
and a quiet edit hands them a plan that is not the one they drew.

### 2. `:root` is rewritten; the plan renders in a shadow root

A plan's `<style>` declares its palette on `:root`. Inlined into the app,
`:root` is the document root, so an uploaded plan restyles the whole page
through it. The normalizer rewrites `:root` to `svg`, and `FloorPlanMap`
attaches the plan to a shadow root.

Both are needed. Shadow DOM alone would break the plan's own colours —
`:root` matches nothing inside a shadow tree. The rewrite alone would leave
generic class selectors (`.room`, `.label`) colliding with the app.

Verified on the real ground-floor plan: inside a shadow root the normalized
output resolves `--wall` to `rgb(37,49,60)` and `.opd` to `rgb(220,236,255)`,
while the host page's own text colour is unchanged.

### 3. The floor owns the coordinate space

Every `place.x/y` and `nav_node.x/y` is a point in the plan's viewBox.
A plan with a different viewBox would move every pin and every route with no
error anywhere — the worst failure shape available here.

So `floor.viewbox` is the contract: the first plan for a floor defines it,
and every later upload must match or is rejected. Re-mapping coordinates is
separate work, not a side effect of an upload.

### 4. What fails the upload, and what only warns

Rejected, because the plan would be wrong for everyone: a non-SVG root, a
missing or mismatched viewBox, anything off the allowlist, more than one
`data-floor` group or one naming a different floor, and a missing
`#route-layer`.

Warned, because the app already has an honest degraded state for it: a place
or navigation node the model holds but the plan does not draw, and a place
the plan draws that the model does not hold. Refusing a whole floor over one
undrawn room would be worse than showing the rest of it. Warnings are
returned on upload and stored with the plan so the staff console can show
what the current plan does not cover.

### 5. Plans are append-only, content-addressed, and stored in Postgres

`floor_plan` rows are never updated or deleted; `floor.active_plan_id` points
at the live one, so a rollback is a pointer move. The row carries both the
normalized SVG (served) and the uploaded original (ADMIN download only,
never sent to a patient), plus `created_by` — which makes the table its own
upload audit.

Postgres rather than object storage, primarily because an upload must be
validated against `place` and `nav_node` rows and committed with them in one
transaction; splitting the bytes into a separate store would make that a
two-phase write that cannot roll back. That the real plans are 12 KB and
7 KB, and that compose runs no object store, are secondary.

**Migration trigger:** move to object storage when plans exceed ~1 MB, or on
the day raster content is allowed.

### 6. Serving: immutable asset, revalidating pointer

```
GET /api/v1/floors                              → [{ floorId, code, levelOrder, viewBox, planUrl }]
    Cache-Control: no-cache + ETag
GET /api/v1/floors/{floorId}/plan/{sha256}.svg
    Cache-Control: public, max-age=31536000, immutable
    Content-Type: image/svg+xml · X-Content-Type-Options: nosniff
    Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'
```

The digest is in the URL, so the large stable asset is fetched once and never
revalidated; only the small pointer document is. The CSP and `nosniff` cover
the case of that URL being opened directly, where an SVG would otherwise run
script in the API's origin — a second line behind the normalizer, not a
substitute for it.

### 7. The navigation graph moves to the API in the same change

`GET /api/v1/navigation/nodes` and `/edges` (the `Service` methods already
exist; only handlers are missing) replace `features/floorplan/graphs.ts`, and
`GET /api/v1/floors` replaces `FLOOR_CODES` and `floorPlanFor`.

Migrating only the SVG is not an option: the two sources would drift in
opposite directions, and the drift leaves the screen — it gets printed onto
QR stickers and stuck to walls.

This does not merge the two models. The navigation graph stays authored data
with its own tables and its own module (ADR-0002); what changes is only which
copy the web app reads.

### 8. The graph is still not editable through the UI

This ADR moves where the graph is *read from*. Editing it through a screen
needs a validator that runs before commit — connectivity from every entrance
to every place entry, the same check again under `accessibleOnly` (#99 ships
stair avoidance, so a graph that leaves only stairs strands wheelchair
users), bidirectional edge pairs, positive distances, no orphans — and that
is its own decision. Until then the staff console gets a read-only graph
health view, which costs a fraction and still tells an admin what is broken.

### 9. Patients are not pinned to a plan version

A plan changes because the building changed; a patient holding the old one is
being actively misdirected, which is worse than a surprising refresh. The
immutable-URL design already makes the change gentle — an in-flight patient
keeps their cached copy until they next read the pointer.

### 10. `packages/floorplans/` stays, demoted

It remains the seed for the first upload (a migration inserts the two plans
as the initial `floor_plan` rows), the test fixtures, and the README that
documents the drawing conventions. Nothing under `apps/web` imports it at
runtime any more.

## Consequences

### Positive

- Admin-configurable map data becomes possible at all, which is FR-11's
  premise.
- The patient app stops shipping a copy of data the API already owns; plan
  and graph have one source of truth again.
- Immutable, content-addressed assets make the plan cacheable forever and a
  rollback a pointer move.
- The normalizer and its corpus are a security asset the repo did not have:
  every payload class that could reach a patient's DOM now has a test.

### Trade-offs

- The plan becomes a network fetch where it used to be in the bundle. Mitigated
  by prefetching it from the journey screen and by rendering turn-by-turn
  text — which comes from the route API, not the plan — without waiting for
  it, so a slow or failed fetch degrades to directions rather than a dead
  screen.
- The allowlist is closed, so a plan drawn with gradients, filters, patterns
  or `<use>` is rejected until those are deliberately added. Accepted: the
  MVP plans need none of them, and each one is url() surface.
- CSS may not contain `<` or `&`. This removes any dependence on how a
  browser tokenizes `<style>` inside inlined foreign content, at the cost of
  CSS nesting's `&`, which the plans do not use.
- Admin-set service point names still do not translate — ADR-0012 §4's
  trigger fires for `name` only, and its documented fallback (client catalog
  by code, else the server's `name`) covers it. A server-side i18n table
  stays a separate decision, taken when an admin-created code actually needs
  English.

### Explicitly not decided here

Raster/scanned plans, editing the navigation graph through a UI, re-mapping
coordinates when a floor is redrawn, and multi-building/multi-hospital
tenancy.
