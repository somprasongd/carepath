# AGENTS.md — apps/web

Guidance for AI assistants working inside the web app. The repo-root
[AGENTS.md](../../AGENTS.md) still applies (architecture boundaries, contract
rules). Structure overview: [docs/architecture/web-app.md](../../docs/architecture/web-app.md).
Visual spec: [DESIGN.md](../../DESIGN.md) at the repo root.

## Commands

```bash
npm run dev        # vite dev server (root: npm run dev:web)
npm run build      # tsc -b && vite build — must pass before committing
npm run lint       # oxlint (root: npm run lint:web)
npm run test       # vitest run (root: npm run test:web); npm run test:watch to re-run on change
npm run gen:api    # regenerate src/api/schema.d.ts from packages/contracts
```

Run `npm run gen:api` whenever `packages/contracts/openapi/carepath.yaml`
changes, and commit the regenerated `schema.d.ts` together with it.

## Linting note

The linter is **oxlint**, not eslint + typescript-eslint: this app builds
with TypeScript 7, whose npm package no longer ships the JS compiler API
that @typescript-eslint's parser requires. oxlint parses TS natively and
covers the correctness rules plus react-hooks rules (`rules-of-hooks`,
`exhaustive-effect-dependencies`). Config lives in `.oxlintrc.json`;
generated files (`routeTree.gen.ts`, `src/api/schema.d.ts`) are ignored
there.

## Testing note

Unit tests run on vitest (`vitest.config.ts`, node environment, `@` alias
mirrored from vite.config.ts). Colocate `*.test.ts` next to the code it
tests. The highest-value targets are the pure mapping functions in
`features/visit/journey.ts` — they translate HIS statuses into what
patients see, so a silent regression there is a UX bug, not a crash.

## Working rules

- **Types come from the contract.** Never hand-write a shape that exists in
  `src/api/schema.d.ts`. Request/response changes belong in the contract
  first (integration-boundary change — check `apps/api` and `apps/mock-his`
  for drift, per the root AGENTS.md).
- **Server state = TanStack Query** in `features/*/queries.ts`. Screens never
  call `fetch` directly; add or extend a hook instead.
- **`design-system/` stays API-ignorant.** It renders props. Anything that
  knows a domain concept (visit, service point) goes in `features/`.
- **Tokens, not hex.** Every colour/radius/size lives in the `@theme` blocks
  of `src/styles/index.css`. A hex value in a screen means a token is
  missing — add one there instead. There is no `tailwind.config.js`.
- **Routing is file-based.** `src/routeTree.gen.ts` is generated — don't edit
  it. Files in `routes/` starting with `-` are colocated components, not
  routes. Read-only data parameters (like `?visit=`) are declared with
  `validateSearch` on the route.
- **Tokens never touch a screen.** The staff access token lives in memory in
  the auth context and the refresh token in `localStorage` (ADR-0010); only
  `api/client.ts` reads them, attaches them, and performs the single retrying
  refresh on a 401. Never put a token in a query key, a URL, a log, or JSX.
  Role checks in the UI hide affordances — the API is what enforces them.
- **Demo data is scoped.** `src/mocks/demo-data.ts` feeds only `/design` and
  the staff screens (no endpoints yet). Patient screens must render live
  visit data via `features/visit`; when a staff endpoint lands, move that
  screen to a query and shrink the demo file.

## Design rules that are easy to break

These come from DESIGN.md and are load-bearing, not stylistic:

- Exactly **one primary (orange) action per patient screen**. Orange means
  "your route, right now" — never decorative, never a staff action.
- Dark `ink` text on orange fills — never white (fails WCAG AA on this orange).
- Patients never see domain vocabulary (`ServicePoint`, `READY`,
  `serviceCode`). Translate to plain Thai in `features/visit/journey.ts`.
- Don't fake capability: a place without a navigation node shows the
  "ยังไม่รองรับเส้นทาง" state — never a route line or routable badge.
- `StickyActionBar` requires its single primary action as children; when
  nothing is actionable, render no bar at all.
