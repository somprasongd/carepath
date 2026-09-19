# ADR-0007: Hexagonal Module Layout with Transaction-in-Context

- Status: Accepted
- Date: 2026-09-19

## Context

`apps/api` grew as a single `main.go`: HIS proxying, the next-step rule, and a hardcoded service-point map all in one file. The API now needs PostgreSQL (`DATABASE_URL` was already wired in compose), and upcoming modules (journey, navigation, location) will share data across module boundaries. We need a structure that keeps ADR-0001's modular monolith honest: explicit module boundaries, business logic away from transport and SQL, and a way for one use case to write through several modules atomically.

## Decision

Structure `apps/api` as a hexagonal (ports & adapters) modular monolith:

1. **Each module** (currently `visit`, `servicepoint`, `his`) owns its layers as files within one package: `handler.go` (HTTP), `service.go` (business logic + the module's public `Service` interface), and a persistence port (`Repo`) implemented by a `postgres` adapter subpackage. Shared infrastructure lives in `internal/platform/db`.
2. **Dependencies point inward**: handler → service → (repo port) ← postgres adapter. `cmd/server/main.go` is only the composition root that wires adapters to ports.
3. **Transactions travel in `context.Context`.** `db.Transactor.WithinTransaction(ctx, fn)` opens a `pgx.Tx`, stores it in ctx, and commits/rolls back around fn. Repos get their connection via `db.Querier(ctx)` — the ambient transaction if present, otherwise the pool. A nested `WithinTransaction` call joins the ambient transaction instead of opening a new one.
4. **Cross-module calls go through the callee's `Service` interface only, never its `Repo`.** Because the service methods receive and forward the caller's ctx, a call made inside a transaction transparently joins it. The HIS is an external system and never participates in database transactions.
5. **Database access** uses `pgx/v5` + `pgxpool` with hand-written SQL — no ORM, no codegen.
6. **Schema changes use golang-migrate** with SQL files in `infra/postgres/migrations/`. A one-shot `migrate` service in docker compose runs before `api` (`service_completed_successfully`); the application itself never runs migrations. This replaces the old `init.sql` bootstrap mount.

Go cannot forbid importing another module's `Repo` at compile time; the rule is enforced by convention, package docs, and review (a linter such as depguard can be added later if drift appears).

## Consequences

### Positive
- New modules (journey, navigation, location) slot in with the same shape: define `Service` + `Repo`, add a postgres adapter, wire in main.
- Use cases spanning several modules stay atomic — the transaction boundary is owned by the orchestrating service, not by repositories or handlers.
- Swapping Mock HIS for a real HIS (ADR-0005) touches only the `his` module's adapter.

### Trade-offs
- The ctx carries an hidden transaction; a repo called outside a transaction silently uses the pool instead of failing. Unit tests assert the propagation contract.
- Joining nested transactions means no partial rollback (no savepoints) — acceptable for the MVP's small use cases.
- Module discipline relies on review rather than the compiler.
