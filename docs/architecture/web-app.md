# Web App Architecture

Canonical description of the `apps/web` structure: how the code is organized, where data comes from, and which conventions keep it that way. Visual language lives in [DESIGN.md](../../DESIGN.md) (repo root); backend structure in [technical-blueprint.md](technical-blueprint.md).

## Stack

- React 19 + TypeScript, built with Vite (`rolldown-vite`)
- Tailwind CSS v4, CSS-first configuration — there is **no `tailwind.config.js`**; tokens are declared in `src/styles/index.css` (`:root` palette, `@theme`, `@theme inline`) and everything else reads Tailwind utilities
- shadcn/ui — generated components live in `src/design-system/ui/`, edited in place to carry CarePath variants; semantic shadcn names (`--background`, `--card`, …) are mapped onto the CarePath palette in `styles/index.css`
- TanStack Router (file-based routing, route generation via `@tanstack/router-plugin` in `vite.config.ts`)
- TanStack Query for server state

## Directory map

```
apps/web/src/
  routes/            File-based routing — one file per URL
    __root.tsx       Base font/colour only, no chrome
    index.tsx        Redirects / → /login (the staff console is the front
                     door; patients enter by naming their VN, not by default)
    login.tsx        Staff/admin login (patients never see this — LINE only)
    patient/         journey, navigate (?visit=<VN> search param — the VN
                     entry screen asks for it when absent; no demo default)
    staff/           overview, service-points, patients, floor-plan
                     (placeholder), queue*, pathway-templates* (`*` = screen
                     still reads mocks/demo-data.ts)
    shared.tsx       Relative's read-only view (#90) — token in the URL
                     fragment, no auth providers, no menu
    design.tsx       The living DESIGN.md catalogue (components + screens)
    -design/         Components for the design page (`-` prefix = not a route)
  design-system/     Presentation primitives — must stay ignorant of the API
    ui/              shadcn CLI output, edited in place
    tokens.ts, …     CarePath-authored primitives (Button, Card, JourneyRail, …)
  auth/              Both auth surfaces (ADR-0010): AuthProvider, LoginGate,
                     RequireAuth, DemoAuthProvider, LiffAuthProvider for
                     `/patient/*` (LINE/demo identity) · StaffAuthProvider,
                     StaffAuthContext, RequireStaffAuth for `/staff/*` (JWT)
  features/          Domain compositions — where the API is allowed in
    visit/           queries.ts (useVisit), journey.ts (VisitView → JourneyRail),
                     staff.ts (staff visit list + transitions), components/
    servicepoint/    queries.ts — live GET /api/v1/service-points
    analytics/       queries.ts — GET /api/v1/analytics/overview (15s polling),
                     overview.ts (aggregate → cards/rows; null renders as "—",
                     never 0 — NFR-10)
    navigation/      queries.ts (useNavigationRoute), route.ts (polylines +
                     turn-by-turn cues from the route response)
    floorplan/       destination.ts, plans.ts — SVG floor-plan lookups
    share/           queries.ts (mint/revoke/read, constant query key),
                     shared-view.ts (SharedJourney → Thai labels, screen
                     model), components/ShareSheet.tsx (patient-side sheet)
    queue/           components only — no query layer yet (queue is still demo)
  api/               client.ts (fetch wrapper, ApiError, token attach + refresh)
                     + schema.d.ts (generated)
  mocks/             demo-data.ts — static data for /design and the screens
                     still marked `*` above (queue, pathway-templates)
  styles/            index.css — the single token source (@theme)
```

Files in `routes/` whose name starts with `-` are colocated components excluded from routing. `routeTree.gen.ts` is generated — never hand-edit it.

## Data flow

```
packages/contracts/openapi/carepath.yaml
  → (npm run gen:api) → src/api/schema.d.ts
  → src/api/client.ts (fetch, same-origin /api — Vite proxy in dev, nginx edge in prod; VITE_API_BASE_URL only overrides the origin)
  → features/visit/queries.ts (TanStack Query, queryKey ['visit', id])
  → features/visit/journey.ts (mapping to design-system types)
  → routes/patient/* screens
```

Status per screen today:

| Screen | Data |
| --- | --- |
| `/patient/journey` | Live — `GET /api/v1/journeys/:visitId`; requires `?visit=<VN>` — without it the VN entry screen (`-VisitEntryScreen`) asks first (the old VISIT-001 demo default is gone); polls every 15s |
| `/patient/navigate` | Live end-to-end — floor plan from `packages/floorplans`, route + turn-by-turn cues from `GET /api/v1/navigation/route` (`features/navigation`). QR-based current-location is API-only; the screen still shows a static "scan the QR at the service point" instruction, no camera/scanner wired in |
| `/staff/patients`, `/staff/service-points` | Live (`/api/v1/staff/visits` + transitions, `/api/v1/service-points`) |
| `/staff/overview` | Live — `GET /api/v1/analytics/overview?window=today` (`features/analytics`); polls every 15s; EXECUTIVE role only (ADR-0010), a metric with no data yet renders as "—" |
| `/staff/queue`, `/staff/pathway-templates` | Still `mocks/demo-data.ts` — no backing endpoint exists yet (queue has no call-next API at all) |
| `/login` | Live — real `POST /api/v1/auth/login` (ADR-0010); no role picker, the landing screen is derived from the role in the returned token |
| `/shared` | Live — `GET /api/v1/shared/journey` with the share token from the URL fragment (ADR-0011); polls every 15s, stops on DONE. No auth providers mount here — the share token is the only credential this surface ever holds |
| `/design` | Demo data by design; it must never depend on the API |

The whole `/staff/*` group mounts behind `RequireStaffAuth` and every request carries a staff access token, regardless of whether that individual screen's data is live or still demo.

## Commands

```bash
npm run dev        # vite dev server (from apps/web; root: npm run dev:web)
npm run build      # tsc -b && vite build (root: npm run build:web)
npm run lint       # oxlint (root: npm run lint:web)
npm run test       # vitest run (root: npm run test:web)
npm run gen:api    # regenerate src/api/schema.d.ts from the contract
```

Linting uses oxlint rather than eslint + typescript-eslint: the app builds with TypeScript 7, whose npm package no longer ships the JS compiler API that @typescript-eslint's parser needs. Unit tests are vitest, colocated as `*.test.ts` — the mapping functions in `features/visit/journey.ts` are the priority coverage target. `gen:api` pins `openapi-typescript` + `typescript@5` in an isolated npx run for the same TS7 reason — see [packages/contracts/README.md](../../packages/contracts/README.md).

## Staff authentication (ADR-0010)

Two auth surfaces coexist and must not be merged: `AuthProvider`/`LoginGate`/
`RequireAuth` gate `/patient/*` on a LINE (or demo) identity; `StaffAuthProvider`/
`RequireStaffAuth` gate `/staff/*` on a username/password login. Both live under
`src/auth/`, but each surface keeps its own token store and context.

- **The access token lives in memory only** (React context state) — it is on
  every request, so keeping it out of `localStorage` shrinks the XSS payoff.
- **The refresh token lives in `localStorage`** — that is what survives a page
  reload without a second login. The trade-off (and why an httpOnly cookie was
  rejected for the MVP) is recorded in ADR-0010 §12.
- `api/client.ts` owns the retry: attach the access token, and on a 401 attempt
  exactly one refresh, retry the original request once, then clear both tokens
  and route to `/login`. One refresh in flight at a time — concurrent 401s wait
  on the same promise, never fire N refreshes.
- `setApiAuthToken` currently holds **one** module-level token, set by the
  patient auth providers. The staff token must not reuse that slot: both route
  trees live in the same SPA, so one shared slot can send a patient session
  token to a staff endpoint. One slot per audience. The share token
  (`setApiShareToken`, ADR-0011) is a third slot with a fourth rule: it is the
  only bearer `/shared` sends, it is read once from the URL fragment on mount,
  and it is never attached anywhere else.
- Never render a token, never log one, never put one in a URL or a query key —
  the URL-fragment exception is the share token's arrival on `/shared`, and it
  is the only one (browsers do not send fragments to servers).
- Role gating is UI affordance, not security: hiding an admin action in React
  is a courtesy; the API is what actually enforces it.

## Conventions

- A response shape that exists in `schema.d.ts` is never hand-written; change the contract and regenerate (contract changes are integration-boundary changes — see the root AGENTS.md rules).
- Server state goes through TanStack Query in `features/*/queries.ts`; screens never call `fetch` directly.
- No hex values in screens — if one appears, a token is missing from `styles/index.css`.
- Zone colours stay byte-identical to the `:root` block in `packages/floorplans/floors/*.svg`.
- `design-system/` renders props only; anything that knows a domain concept belongs in `features/`.
- A screen never decides what a user may do from a locally stored flag — read the roles out of the auth context, which reads them from the token.
- Agents working in this app should read `apps/web/AGENTS.md` for the working rules.
