# CLAUDE.md

Guidance for Claude Code (and any AI assistant) working in this repository.

## What this is

CarePath is a patient journey and indoor navigation platform for hospitals, sitting above the HIS. It combines care/service flow, hospital spatial data, indoor routing, and current-location providers (QR, Zigbee). It started as a hackathon MVP — favor small, working increments over speculative architecture.

Core mental model (see `README.md` for the full picture):
- **HIS** knows what service the patient needs.
- **Care Graph** knows what the next step is.
- **ServicePoint** links a care step to a physical place.
- **Hospital Map** knows where that place is.
- **Navigation Graph** knows how to get there.
- **Location Provider** knows where the patient is now.
- **LINE LIFF/Web** presents the journey and route to the patient.

## Repository layout

```
apps/
  api/            Go + Fiber v3 — CarePath backend API
  mock-his/       Go + Fiber v3 — mock HIS service for local dev/demo
  web/            React + TypeScript + Vite — patient/admin web app (own AGENTS.md)
packages/
  contracts/      Shared OpenAPI contracts (source of truth for API shape)
  floorplans/     SVG floor plans (I-1301 ground, I-1302 upper) + navigation graphs
infra/
  docker/         Dockerfiles per service
  postgres/       DB schema migrations (golang-migrate)
docs/             Architecture, requirements, ADRs, integration notes, API docs — see docs/README.md
```

Go services share a workspace via `go.work` (`apps/api`, `apps/mock-his`). Web app is an npm workspace (`apps/web`), declared in the root `package.json`.

Nested instructions: `apps/web/AGENTS.md` holds the web app's working rules (contract typegen, TanStack Query, design-system conventions) — read it before changing anything under `apps/web`. Web app structure is documented in `docs/architecture/web-app.md`; the `apps/api` internal module structure in `docs/architecture/technical-blueprint.md`.

## Common commands

```bash
# Everything via Docker Compose
cp .env.example .env
docker compose up --build      # web :5173, api :8080, mock-his :8090, postgres :5432

# Or individually (Makefile)
make api        # go run ./cmd/server in apps/api
make mock-his   # go run ./cmd/server in apps/mock-his
make web        # npm run dev in apps/web
make fmt        # gofmt -w on apps/api and apps/mock-his
make up / down / logs   # docker compose lifecycle

# Go
go work sync
cd apps/api && go run ./cmd/server
cd apps/mock-his && go run ./cmd/server

# Web (from root)
npm run dev:web
npm run build:web
npm run lint:web  # oxlint (apps/web/AGENTS.md says why not eslint)
npm run test:web  # vitest
npm run gen:api   # regenerate apps/web/src/api/schema.d.ts from packages/contracts
```

CI runs on pushes/PRs to `main` (`.github/workflows/ci.yml`): Go fmt/vet/build/test per module, plus web lint, test, and build. There are no E2E/browser tests yet — don't rely on CI alone to catch UI regressions; verify patient flows in the browser when changing screens.

## Architecture boundaries — do not violate

These come from the ADRs in `docs/adr/` and are load-bearing:

1. CarePath does **not** read the HIS database directly (ADR-0005). All HIS access goes through an internal adapter/port; Mock HIS implements the same conceptual contract and is meant to be swapped for a real HIS adapter later.
2. Care Graph and Navigation Graph are **separate models** (ADR-0002) — clinical/operational flow vs. physical wayfinding. Don't couple them.
3. Floor plans are SVG-based for the MVP (ADR-0003) — no 3D engine work.
4. Location is abstracted behind a provider interface (ADR-0004): QR is baseline, Zigbee is optional/phase 2, manual selection is fallback/debug. New location sources implement this interface rather than being special-cased.
5. API contracts between CarePath and Mock HIS are REST/JSON, defined in version-controlled OpenAPI under `packages/contracts/openapi/` (ADR-0006). Treat that directory as the source of truth for request/response shapes — don't let handlers drift from it silently.
6. Keep the backend a modular monolith (ADR-0001) rather than splitting into microservices; monorepo is intentional for hackathon-speed coordination.
7. Patient identity and staff identity are **separate mechanisms** (ADR-0010): patients get an opaque session token from a verified LINE identity (`internal/session`), staff/admin get an argon2id password login returning a JWT access token plus a rotating refresh token (`internal/auth`). Neither token is accepted by the other's surface, and the two must not be merged into one table or one middleware.

If a change would cross one of these boundaries, flag it and check whether it needs a new/updated ADR in `docs/adr/` before implementing.

## Working conventions

- Read `docs/README.md` first for the documentation index; `docs/requirements/*` for product scope, `docs/architecture/*` for how the system is organized, `docs/adr/*` for why a decision was made.
- Before working in `apps/api`, read `apps/api/AGENTS.md` — it defines how to create a module and the DB-transaction, logging, and error rules.
- For non-trivial changes, use Plan Mode first: read the relevant docs/ADRs, produce a short plan (files to touch, sequence, risks), and get it reviewed before implementing. See `docs/process/ai-native-sdlc-workflow.md`.
- Go code: format with `gofmt` (`make fmt`) before committing.
- Changing `packages/contracts/openapi/*.yaml` is an integration-boundary change — call it out explicitly in the PR description and check both `apps/api` and `apps/mock-his` for drift.
- Keep documentation and code in sync: a new ADR-worthy decision belongs in `docs/adr/`, not just in a commit message.

## Common mistakes (update this section when Claude repeats one)

- (none recorded yet)
