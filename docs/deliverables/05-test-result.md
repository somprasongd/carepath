# 5. Test Result (ผลการทดสอบ)

**Deliverable 5 of 6** per the hackathon brief §10: a table of at least 10 test cases with pass/fail results and what was fixed. Every result below was actually executed during this pass, not written from memory — see [3.4 Test environment](#test-environment) for how.

## Summary

| Suite | Tests | Result |
|---|---|---|
| `apps/api` (Go, `go test ./...`) | 22 | 22 passed, 0 failed, 0 skipped |
| `apps/mock-his` (Go, `go test ./...`) | 6 | 6 passed, 0 failed |
| `apps/web` (Vitest, `npm test`) | 18 | 18 passed, 0 failed |
| **Automated total** | **46** | **all passing** |

Plus 16 manual/live test cases below (API calls against the running services, and one browser check of the actual web app) — **1 real defect found and fixed** during this pass (TC-05).

## Test environment

- `apps/api` and `apps/mock-his` run locally with `go run ./cmd/server`; `apps/web` via `npm run dev` (Vite, port 5173).
- Postgres: a disposable `postgres:17-alpine` container (matching the version pinned in `docker-compose.yml`), migrated with the two real migrations in `infra/postgres/migrations/` — the same schema `docker compose up` would produce.
- Demo data: Mock HIS's built-in `VISIT-001` (patient `PATIENT-DEMO-001`), and the 3 seeded service points (`REGISTRATION`, `LAB`, `PHARMACY`).
- Note on scope: `apps/mock-his` gained a real `internal/mockhis` package and its own test suite (6 tests, covering the ADR-0008 canonical event/command contract) from concurrent work landing in this same checkout while this test pass was running. Its automated suite is included above; its new event/command endpoints were not individually exercised as manual test cases below — only what's listed was actually driven by hand.

## Test cases

| # | Test case | Steps / input | Expected | Actual | Result | Fixed |
|---|---|---|---|---|---|---|
| TC-01 | Get normalized visit view (happy path) | `GET /api/v1/visits/VISIT-001` | 200, visit + all 5 steps + resolved `next` | 200, exact match, `next` resolved to LAB/Laboratory | **Pass** | — |
| TC-02 | Get next actionable step | `GET /api/v1/visits/VISIT-001/next` | 200, first `READY` step resolved to its `ServicePoint` | 200, `{sequence:4, status:READY, servicePoint: SP-LAB}` | **Pass** | — |
| TC-03 | Unknown visit via CarePath API | `GET /api/v1/visits/NOPE` | 404, JSON error body | `404 {"error":"visit not found"}` | **Pass** | — |
| TC-04 | Unknown visit via Mock HIS directly | `GET :8090/api/v1/visits/NOPE` | 404, JSON error body | `404 {"error":"visit not found"}` | **Pass** | — |
| TC-05 | Upstream HIS unavailable | Stop Mock HIS, then `GET /api/v1/visits/VISIT-001` | 502, **generic** client-facing message (NFR-10: no raw system errors); full cause in server log | **First run: FAIL** — client received `"HIS unavailable: Get \"http://localhost:8090/...\": dial tcp [::1]:8090: connect: connection refused"` (raw Go error leaked). Root cause: `httpx.Error` only masked `KindInternal`, not `KindUpstream`, even though both are 5xx and both wrap a raw cause. | **Fail → Fixed → Pass** | Yes — [`httpx.go`](../../apps/api/internal/platform/httpx/httpx.go): mask any status ≥ 500 to a kind-specific generic message (`"upstream service unavailable"` for 502); full detail still logged server-side. Added a regression assertion to `TestErrorStatusPerKind` (`httpx_test.go`) so the leak can't return silently. Re-verified live: client now gets `{"error":"upstream service unavailable"}`, server log keeps the real dial error. |
| TC-06 | SQL-injection-shaped input | `GET /api/v1/visits/VISIT-001%27%3B%20DROP%20TABLE...` | No SQL executed; normal 404, no data loss | 404 `visit not found`; `carepath.service_point` row count unchanged (3) after the attempt. `servicepoint/postgres/repo.go` uses `$1` pgx parameter binding, not string interpolation, for every query. | **Pass** | — |
| TC-07 | Response time budget (NFR-05, ≤3s) | Timed `GET /api/v1/visits/VISIT-001` | < 3000ms | ~3–7ms measured | **Pass** | — |
| TC-08 | `apps/api` automated suite | `go test ./...` with `DATABASE_URL` set (no skips) | All pass, including the 4 Postgres-backed `servicepoint` tests that skip without a DB | 22/22 pass, 0 skipped | **Pass** | — |
| TC-09 | `apps/mock-his` automated suite | `go test ./...` | All pass | 6/6 pass | **Pass** | — |
| TC-10 | `apps/web` automated suite | `npm test` (Vitest) | All pass | 18/18 pass | **Pass** | — |
| TC-11 | Navigation route endpoint (not yet built) | `GET /api/v1/navigation/route?from=A&to=B` | Documented in OpenAPI as "not implemented yet"; should not 500 | Fiber default `404 Not Found` (no route registered) — consistent with the spec's own note, but should eventually be a clean `501`/typed error once the module exists rather than a bare framework 404 | **Fail (expected gap)** | No — tracked as a `navigation` module task, not a bug in existing code |
| TC-12 | Web app renders the patient journey end-to-end | Load `http://localhost:5173` against the live API + Mock HIS | Journey timeline shows registration/screening/doctor completed, blood draw in progress at Laboratory·LAB-01, pharmacy next; no console errors | Matches exactly; 0 console errors | **Pass** | — |
| TC-13 | Graceful handling of an unbuilt navigation target | Click "นำทางไปเจาะเลือด" (navigate to blood draw) in the web app | No crash; a clear, human-readable message (NFR-10) | Shows: "ระบบยังไม่รองรับเส้นทางในอาคารสำหรับจุดบริการนี้ — โปรดถามเจ้าหน้าที่ที่จุดรับลงทะเบียน" — no crash, no raw error | **Pass** | — |
| TC-14 | CORS headers on API responses | Inspect response headers of `GET /api/v1/visits/VISIT-001` | `Access-Control-Allow-Origin: *` present (browser frontend needs this) | Present, plus `Allow-Headers`/`Allow-Methods` | **Pass** | — |
| TC-15 | Brief scenario 1 — full diabetic-patient pathway (§13): registration → screening → blood draw before 11:00 → wait → doctor (needs lab result) → cashier → pharmacy, across 3 buildings | Attempt via current system | End-to-end pathway with prerequisite ordering, registration, and routing across buildings | Not runnable: no registration screen, no Pathway Template, no cross-building navigation graph yet (M1 partial, M2/M3 not built) | **Fail (expected gap)** | No — needs the `hospitalmap`, `journey`/pathway, and `navigation` modules from the [technical blueprint](../architecture/technical-blueprint.md); tracked against [FR-13/FR-14](../requirements/functional-requirements.md) |
| TC-16 | Brief scenario 2 — edge cases (§13): queue-deadline reprioritization, mid-visit unplanned X-ray insertion, elevator-down rerouting, wheelchair avoid-stairs, language switch | Attempt via current system | Each case handled without breaking the existing plan | Not runnable: queue (M7), unplanned-step insertion (M7), routing/accessibility (M5/M6/S5), and i18n (S4) aren't built yet | **Fail (expected gap)** | No — see [deliverable 1](01-requirement-specification.md) §1.3 for the FR/priority each piece maps to |

## What this shows

- Everything **currently built** (visit lookup, next-step resolution, error handling, the patient journey screen) works correctly, including under adversarial input (TC-06) and dependency failure (TC-05).
- The one real defect found (TC-05, an information-disclosure/usability gap) was fixed in the same pass, with a regression test added so it can't silently reappear.
- The two scenario-level failures (TC-15, TC-16) aren't bugs — they're the brief's own acceptance scenarios failing because the underlying Must-Have modules (pathway templates, registration, queue, navigation, i18n) aren't implemented yet, consistent with the gaps already flagged in [deliverable 1](01-requirement-specification.md) and [deliverable 3](03-prototype-wireframe.md). Closing M2/M3/M7 is what turns TC-15/TC-16 from fail to pass.
