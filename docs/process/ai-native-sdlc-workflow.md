# AI-Native SDLC Workflow for CarePath

Source concept: [The AI-Native SDLC Playbook](https://claude.com/blog/the-ai-native-sdlc-playbook) (Claude). This doc adapts that six-stage playbook to CarePath's actual state: a hackathon-born MVP monorepo (Go + Fiber v3 services, a React/Vite web app, shared OpenAPI contracts) with strong requirements/architecture docs and ADRs, a GitHub issue board (epics + MVP issues) driving the backlog, CI on PRs to `main`, a first layer of automated tests, and root `AGENTS.md` guidance — but no contract tests, no `REVIEW.md`, and no skills/hooks yet.

The playbook's core idea: each SDLC stage produces a versioned markdown artifact the next stage reads, so intent → design → plan → code → review → incident forms one auditable loop instead of siloed handoffs. We keep that idea but right-size the governance — this is an MVP team, not a regulated enterprise, so Stage 5/6 gates start light and grow only when they earn their cost.

## Where this repo already stands vs. the playbook

| Stage | Playbook artifact | What CarePath already has | Gap to close |
|---|---|---|---|
| 1. Plan | `intent.md` | `docs/requirements/*` (product/functional/NFR/MVP scope), GitHub issue board (epics + MVP issues) | Board issues capture per-feature intent; no `docs/intent/` files — see "Working the issue board" below |
| 2. Design | `spec.md` + skills | `docs/architecture/*`, `docs/adr/*`, per-issue `## Design` sections | ADRs cover big decisions well; no encoded policy skills |
| 3. Build | `plan.md`, CLAUDE.md, skills, hooks | Root + nested `AGENTS.md` (`apps/api`, `apps/web`), Plan Mode convention | No hooks yet |
| 4. Test | feedback loops, evals | Go + Vitest tests (visit journey, API client, platform packages), run by CI | No contract tests against `packages/contracts/openapi`; coverage limited to a few units |
| 5. Deploy | layered review, CI/CD gates | `docker-compose.yml`, `infra/docker/*`, `.github/workflows/ci.yml` (Go fmt/vet/build/test + web lint/test/build) | No `REVIEW.md`, no contract-compatibility gate |
| 6. Maintain | control-band monitoring, security scans | None | Not needed yet at hackathon scale — noted as post-MVP roadmap |

## Working the issue board

The GitHub issue board (epics `#1`–`#10` plus per-feature MVP issues) is the backlog this workflow operates on: the board owns *what to build*, this workflow owns *how each change flows*. One issue = one pass through the artifact chain.

| Stage | Practice with issues |
|---|---|
| 1. Plan | The issue body (scope, acceptance criteria, dependencies) **is** the intent capture — no separate file. Clarify unresolved scope in issue comments before starting work. |
| 2. Design | Only for boundary-crossing issues (contract edits per ADR-0006, state ownership, architecture). The spec is a `## Design` section **in the issue body** — durable decisions that outlive the issue go to `docs/adr/` instead (test: *would other work obey this after the issue closes?*). Example: where visit state lives (HIS vs CarePath DB) needs an ADR before the transition API is built. |
| 3. Build | One issue = one branch + one PR. For non-trivial issues, plan first; the PR description carries the plan (files to touch, sequence, risks, how it will be verified). |
| 4. Test | The issue's acceptance criteria are the test checklist — an issue is not closeable until its tests exist and pass. |
| 5. Deploy | The PR must pass CI; on merge, close the issue with an evidence comment (files/commits proving each criterion, as done for #11/#14/#23). |
| 6. Maintain | Demo bugs and failing flows become new issues, re-entering Stage 1. |

`docs/intent/NNNN-*.md` is therefore **not** used for board work; keep it for cross-cutting or post-MVP proposals that don't have an issue yet.

## Stage 1 — Plan: capture intent per feature

- Default vehicle is the **issue board** — see *Working the issue board* above. A `docs/intent/NNNN-short-name.md` file (numbered like ADRs) is reserved for cross-cutting or post-MVP proposals that don't have an issue yet.
- Workflow: whoever proposes a feature (e.g. "add Zigbee fallback to manual location") talks it through with Claude, which asks clarifying questions and writes a short `intent.md`: problem, why now, constraints, out-of-scope.
- Product/tech lead approves in the PR that adds the file before Stage 2 starts.
- Keep it short — one screen. This is not a replacement for `docs/requirements/*`, which stays the durable source of product scope; `intent.md` is the per-change trigger.

## Stage 2 — Design: fold requirements + design into one pass

A spec is **not** a readiness gate for every issue — only boundary-crossing issues need one (anything editing `packages/contracts/openapi/*.yaml` per ADR-0006, changes to state ownership, or anything touching the architecture boundaries in the root `README.md`). Everything else goes straight from acceptance criteria to the Stage 3 plan.

For board work, the spec lives **in the issue body** as a `## Design` section — edit the body, don't append comments (the body is current state; comments are for discussion). It should cover: affected components (Care Graph / Navigation Graph / ServicePoint / Location Provider — keep ADR-0002 and ADR-0004's boundaries explicit), API contract changes, the chosen approach among alternatives, and open questions.

**Spec in issue vs. ADR in repo** — the dividing line: *would other work have to obey this decision after this issue closes?* If yes, it graduates to a numbered ADR in `docs/adr/` (and the issue's `## Design` section references it); if it's local to this issue, it stays in the issue. Issue bodies aren't version-controlled — an accepted trade-off, safe because anything durable has already moved to an ADR.

If Claude proposes a change that crosses an architecture boundary (e.g. touching HIS access directly, merging Care Graph and Navigation Graph), it must flag it and a new or amended ADR is required before Stage 3 — this is the "policy as skill" check from the playbook, encoded here as an explicit rule rather than a formal skill file for now.

For non-board proposals still living in `docs/intent/NNNN-*.md`, the companion spec is `docs/intent/NNNN-short-name.spec.md` as originally planned.

## Stage 3 — Build: plan before code, then implement

**Root `AGENTS.md`** (added as `CLAUDE.md` at first, since split into a root file plus nested `apps/api/AGENTS.md` and `apps/web/AGENTS.md`) so every session starts with the same institutional knowledge: layout, commands, architecture boundaries, conventions. Update it whenever Claude repeats a mistake.

**Plan Mode convention:**
- For anything non-trivial, start Claude Code in plan mode, point it at the `intent.md`/`spec.md` pair, and let it interview you before touching files.
- Output is a short `plan.md` (can live in the PR description rather than a new file, given repo size): files to touch, sequence, risks, how it'll be verified.
- Only exit plan mode once the plan looks right.

**Parallel work:** the repo already splits cleanly into `apps/api`, `apps/mock-his`, `apps/web`, `packages/contracts`. Independent features in different apps are a natural fit for separate git worktrees / parallel Claude Code sessions, since they rarely touch the same files.

**Hooks (introduce once the pain shows up, not preemptively):** candidates once `.claude/settings.json` hooks are added —
- Run `gofmt`/`goimports` on save for `apps/api` and `apps/mock-his` (matches the existing `make fmt` target).
- Run the web linter/formatter on save for `apps/web`.
- Block direct edits to `packages/contracts/openapi/*.yaml` unless the diff also touches a `docs/adr/*` or `docs/intent/*` file referencing the contract change (contracts are the integration boundary per ADR-0006).

**Skills (introduce as policies stabilize):** the first one worth writing is a "PHI/patient-data handling" skill, since CarePath sits above hospital HIS data — encode what fields count as sensitive, where they may/may not be logged or displayed, and reference ADR-0005 (no direct HIS DB access, must go through the adapter).

## Stage 4 — Test: give Claude a way to check its own work

A first layer of tests exists (visit journey service, web API client, platform packages) and CI runs them. The highest-leverage remaining gap before Stage 5 gating means much is **contract tests**: nothing yet verifies that the `apps/api` and `apps/mock-his` handlers actually satisfy `packages/contracts/openapi/*.yaml`.

- **Go services** (`apps/api`, `apps/mock-his`): add table-driven tests per handler/domain function; `go test ./...` becomes the loop Claude iterates against before you look at a diff.
- **Web** (`apps/web`): add Vitest + React Testing Library; component and routing-flow tests.
- **Contract tests**: validate that `apps/api` and `apps/mock-his` handlers actually satisfy `packages/contracts/openapi/*.yaml` — this is the concrete, automatable version of ADR-0006's intent.
- **Bug-fix pattern**: write the failing test first, have Claude confirm it fails for the right reason, then fix the code without touching the test.
- **Evals**: skip a formal eval harness for now (20-50 task suites are overkill pre-CI); revisit once there's a stable API surface to regress against — e.g. routing correctness for the Navigation Graph is a good future eval candidate.

## Stage 5 — Deploy: start with CI, add gates as the team grows

- **CI exists**: `.github/workflows/ci.yml` runs Go fmt/vet/build/test for both Go modules (`go.work` ties them together) plus web lint/test/build on pushes/PRs to `main`. Extend it with a contract-compatibility gate once contract tests exist (Stage 4).
- **`REVIEW.md`**: define what every PR gets checked for — correctness, adherence to the architecture boundaries in the root README, PHI/security handling, and OpenAPI contract compatibility. Claude can run this pass on every PR; a human reviewer then focuses on intent and risk instead of re-deriving the checklist.
- **Deploy gating**: given `docker-compose.yml`/`infra/docker/*` is the current deploy target (local/demo), there's no production gate yet. When a real staging/prod split exists, apply the playbook's rule directly: autonomous deploys allowed to staging, a named human approval required for anything production-like.

## Stage 6 — Maintain: defer, but note the shape

Not worth building at hackathon/MVP scale — no production traffic, no on-call. When CarePath moves past MVP:
- Recurring security scans make sense early given patient-adjacent data, even before full control-band monitoring exists.
- A lightweight version of the loop: a failing CI run or a reported bug becomes a new GitHub issue (or a `docs/intent/*` entry if it's cross-cutting and not board-sized), re-entering Stage 1 — this costs nothing to adopt now and is worth doing from day one even without automated triggers.

## Artifact chain (adapted)

```
docs/intent/NNNN-name.md        (Stage 1 — problem + why)
docs/intent/NNNN-name.spec.md   (Stage 2 — design, or promoted to docs/adr/ if architectural)
PR description "plan"           (Stage 3 — files, sequence, risks; plan-mode output)
PR diff + CI + REVIEW.md pass   (Stage 4/5 — tests prove it, review pass checks policy)
docs/intent/ (new entry)        (Stage 6 — incidents/bugs loop back to Stage 1)
```

`docs/adr/*` remains the durable "why" record for decisions that outlive a single feature, exactly as it's used today — this workflow doesn't replace it, just adds the lighter-weight per-feature layer around it.

Issue-board variant (how work actually flows today):

```
issue body                       (Stage 1 — scope + acceptance criteria)
issue `## Design` / ADR if boundary crossed   (Stage 2)
PR description "plan"            (Stage 3 — files, sequence, risks)
PR diff + CI + REVIEW.md pass    (Stage 4/5)
issue closed with evidence       (Stage 5)
new issue for demo bugs          (Stage 6 — re-enters Stage 1)
```

## Adoption order

1. **Done**: `AGENTS.md` (root + nested), Plan Mode convention, GitHub issue board as the intent layer, first Go/Vitest tests, CI (`ci.yml`).
2. **Now**: contract tests against `packages/contracts/openapi`, `REVIEW.md`, and grow test coverage with every issue (acceptance criteria = test checklist).
3. **Later**: hooks for formatting/contract-protection, PHI-handling skill.
4. **Post-MVP**: staging/prod deploy gating, security scan cadence, incident→issue loop automation.

Prioritize based on actual friction (per the playbook: "teams prioritize based on their unique bottlenecks rather than sequential rollout") — if requirements churn is the pain, invest in Stage 1/2 first; if broken merges are the pain, jump straight to Stage 5 CI.
