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
    index.tsx        Redirects / → /patient/journey
    patient/         journey, navigate (?visit=<id> search param)
    staff/           overview, service-points, patients*, floor-plan*
    design.tsx       The living DESIGN.md catalogue (components + screens)
    -design/         Components for the design page (`-` prefix = not a route)
  design-system/     Presentation primitives — must stay ignorant of the API
    ui/              shadcn CLI output, edited in place
    tokens.ts, …     CarePath-authored primitives (Button, Card, JourneyRail, …)
  auth/              Patient (LINE LIFF) auth: AuthProvider, LoginGate, RequireAuth
  features/          Domain compositions — where the API is allowed in
    visit/           queries.ts (useVisit), journey.ts (VisitView → JourneyRail),
                     components/AttentionCard.tsx
    servicepoint/    ServicePointRow and friends (demo data for now)
    auth/            Staff/admin auth (ADR-0010) — login/refresh/logout queries,
                     the token store, StaffAuthProvider + RequireStaffAuth
  api/               client.ts (fetch wrapper, ApiError, token attach + refresh)
                     + schema.d.ts (generated)
  mocks/             demo-data.ts — static data for /design and staff screens
  styles/            index.css — the single token source (@theme)
```

Files in `routes/` whose name starts with `-` are colocated components excluded from routing. `routeTree.gen.ts` is generated — never hand-edit it.

## Data flow

```
packages/contracts/openapi/carepath.yaml
  → (npm run gen:api) → src/api/schema.d.ts
  → src/api/client.ts (fetch, VITE_API_BASE_URL)
  → features/visit/queries.ts (TanStack Query, queryKey ['visit', id])
  → features/visit/journey.ts (mapping to design-system types)
  → routes/patient/* screens
```

Status per screen today:

| Screen | Data |
| --- | --- |
| `/patient/journey` | Live — `GET /api/v1/visits/:id` (default `VISIT-001`, override with `?visit=`) |
| `/patient/navigate` | Destination live from the same query; schematic floor plan still demo (only the pharmacy plan exists — other places render the "ยังไม่รองรับเส้นทาง" state until `/api/v1/navigation/route` is implemented) |
| `/staff/*` | Live for the patient list and step transitions (`/api/v1/staff/visits`); queue and overview are still `mocks/demo-data.ts`. Once ADR-0010 lands, the whole group mounts behind `RequireStaffAuth` and every request carries a staff access token |
| `/login` | Currently inert (local state, role cards). Becomes a real `POST /api/v1/auth/login` form under ADR-0010 — no role picker, the landing screen is derived from the role in the token |
| `/design` | Demo data by design; it must never depend on the API |

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

Two auth surfaces coexist and must not be merged: `src/auth/` gates `/patient/*`
on a LINE (or demo) identity; `src/features/auth/` gates `/staff/*` on a
username/password login.

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
  token to a staff endpoint. One slot per audience.
- Never render a token, never log one, never put one in a URL or a query key.
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
