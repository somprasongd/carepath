# AGENTS.md — apps/api

Guidance for AI assistants working in the CarePath API. Read the root `AGENTS.md` first; this file covers API-specific rules. Architecture rationale: [ADR-0007](../../docs/adr/0007-hexagonal-modules-transaction-in-context.md), current structure: "Backend modules" in [technical-blueprint](../../docs/architecture/technical-blueprint.md).

**What this is:** Go + Fiber v3 hexagonal modular monolith. `cmd/server` is the composition root (wiring only). Modules live under `internal/`; shared infrastructure under `internal/platform/`.

## Commands

```bash
go build ./... && go vet ./... && go test ./...   # verify everything before finishing
make fmt                                           # gofmt -w . (also available from repo root)
```

Run from repo root:

```bash
make swag          # regenerate docs/ after adding/changing swag annotations (commit the output)
make migrate-up    # apply infra/postgres/migrations to the compose postgres
make api           # run locally (needs postgres + migrations; default DATABASE_URL targets localhost:5432)
```

Integration tests (`internal/*/postgres/*_test.go`) run against a real Postgres via `DATABASE_URL` and skip when it is unset.

## Creating a new module

A module gets a package only when it has a real feature — never an empty skeleton. Checklist, in order:

1. `internal/<module>/<module>.go` — domain types + the module's `Repo` port interface.
2. `internal/<module>/service.go` — `Service` interface (**the only entry point other modules may call**) + implementation.
3. `internal/<module>/handler.go` — Fiber handlers with swag annotations (`//	@Summary ...`). Omit if the module has no routes.
4. `internal/<module>/postgres/repo.go` — pgx adapter implementing `Repo`; every query goes through `database.Querier(ctx)`.
5. Wire it in `cmd/server/main.go`: repo → service → handler → routes.
6. New table? Add `infra/postgres/migrations/NNNNNN_name.up.sql` + matching `.down.sql`, then `make migrate-up`. Keep DDL idempotent (`IF NOT EXISTS`, `ON CONFLICT`) so existing volumes still migrate.
7. Tests: unit tests for the service using fakes (see `internal/visit/service_test.go`); integration tests for the postgres adapter (see `internal/servicepoint/postgres/repo_test.go`).
8. `make swag` if you added or changed annotated endpoints.

Reference implementations: `servicepoint` (repo + service, no HTTP), `visit` (full stack incl. handler), `his` (outbound integration port + adapter).

## Rules — DB & transactions

- The application never runs migrations; only the compose `migrate` service does.
- Repos hand-write SQL on pgx (`pgxpool`/`pgx.Tx`). No ORM, no query builders, no codegen.
- **Services own transaction boundaries.** Where an atomic use case starts, wrap it: `s.tx.WithinTransaction(ctx, func(ctx context.Context) error { ... })`.
- Transactions travel in `ctx`. A repo must run **every** query through `database.Querier(ctx)` so it joins the ambient transaction when one is open.
- Nested `WithinTransaction` joins the outer transaction — a cross-module `Service` call made inside a transaction shares it through the ctx automatically. Never pass a tx or pool around explicitly.
- Cross-module calls go through the callee's `Service` interface only. Never import another module's `Repo` or its `postgres` adapter.
- The HIS is an external system and never participates in DB transactions.

## Rules — logging

- Use `log/slog` only — never `fmt.Println`, `log.Printf`, or `panic` for diagnostics.
- Get the request-scoped logger with `logger.FromContext(ctx)`; it already carries `request_id`, `method`, `path`. Always pass `ctx` down so lines correlate across layers.
- Levels: `Debug` for service internals, `Warn` for 4xx outcomes, `Error` for 5xx and panics.
- Errors are logged **once**, at the boundary (`httpx.Error` or the logger middleware) — do not re-log the same error in every layer.
- Keep output structured (`log.Info("resolved next step", "visit_id", id)`); never assemble message strings by hand. Format/level come from `LOG_FORMAT` / `LOG_LEVEL` (default JSON / info) — don't hardcode handlers.

## Rules — errors

- Domain errors are module-level variables built with `apperr`:

  ```go
  var ErrNotFound = apperr.New(apperr.KindNotFound, "care step not found")
  ```

- Kinds map 1:1 to statuses — Invalid 400, Unauthorized 401, Forbidden 403, NotFound 404, Conflict 409, Internal 500, Upstream 502. Don't invent per-module status mappings or return `fiber.Map` errors.
- Add context with an explicit kind: `apperr.Wrapf(apperr.KindInternal, err, "module: what failed")`.
- Classify with `apperr.KindOf(err)` (not `errors.Is`) so wrapping never breaks control flow. Unclassified errors are Internal.
- Handlers respond with `return httpx.Error(c, err)` — it picks the status, writes `{"error": msg}`, and logs with the request ID. Internal errors are masked to `"internal server error"` for clients; the real cause stays in logs. Never stringify an internal error into a response yourself.
