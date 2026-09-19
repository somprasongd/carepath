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
  features/          Domain compositions — where the API is allowed in
    visit/           queries.ts (useVisit), journey.ts (VisitView → JourneyRail),
                     components/AttentionCard.tsx
    servicepoint/    ServicePointRow and friends (demo data for now)
  api/               client.ts (fetch wrapper, ApiError) + schema.d.ts (generated)
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
| `/staff/*` | Demo data from `mocks/demo-data.ts` — staff endpoints don't exist yet |
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

## Conventions

- A response shape that exists in `schema.d.ts` is never hand-written; change the contract and regenerate (contract changes are integration-boundary changes — see the root AGENTS.md rules).
- Server state goes through TanStack Query in `features/*/queries.ts`; screens never call `fetch` directly.
- No hex values in screens — if one appears, a token is missing from `styles/index.css`.
- Zone colours stay byte-identical to the `:root` block in `packages/floorplans/floors/*.svg`.
- `design-system/` renders props only; anything that knows a domain concept belongs in `features/`.
- Agents working in this app should read `apps/web/AGENTS.md` for the working rules.
