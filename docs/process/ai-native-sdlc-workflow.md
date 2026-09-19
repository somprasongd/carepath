# AI-Native SDLC Workflow for CarePath

Source concept: [The AI-Native SDLC Playbook](https://claude.com/blog/the-ai-native-sdlc-playbook) (Claude). This doc adapts that six-stage playbook to CarePath's actual state: a hackathon-born MVP monorepo (Go + Fiber v3 services, a React/Vite web app, shared OpenAPI contracts) with strong requirements/architecture docs and ADRs, but no CI, no automated tests, and no `CLAUDE.md`/skills/hooks yet.

The playbook's core idea: each SDLC stage produces a versioned markdown artifact the next stage reads, so intent → design → plan → code → review → incident forms one auditable loop instead of siloed handoffs. We keep that idea but right-size the governance — this is an MVP team, not a regulated enterprise, so Stage 5/6 gates start light and grow only when they earn their cost.

## Where this repo already stands vs. the playbook

| Stage | Playbook artifact | What CarePath already has | Gap to close |
|---|---|---|---|
| 1. Plan | `intent.md` | `docs/requirements/*` (product/functional/NFR/MVP scope) | No lightweight per-feature intent capture; requirements docs are broad, not per-change |
| 2. Design | `spec.md` + skills | `docs/architecture/*`, `docs/adr/*` | ADRs cover big decisions well; no per-feature spec step, no encoded policy skills |
| 3. Build | `plan.md`, CLAUDE.md, skills, hooks | None | No `CLAUDE.md`, no Plan Mode convention, no hooks |
| 4. Test | feedback loops, evals | None | No Go tests, no web tests, no contract tests against `packages/contracts/openapi` |
| 5. Deploy | layered review, CI/CD gates | `docker-compose.yml`, `infra/docker/*` | No CI pipeline (`.github/workflows`), no `REVIEW.md` |
| 6. Maintain | control-band monitoring, security scans | None | Not needed yet at hackathon scale — noted as post-MVP roadmap |

## Stage 1 — Plan: capture intent per feature

- New folder: `docs/intent/NNNN-short-name.md`, numbered like ADRs.
- Workflow: whoever proposes a feature (e.g. "add Zigbee fallback to manual location") talks it through with Claude, which asks clarifying questions and writes a short `intent.md`: problem, why now, constraints, out-of-scope.
- Product/tech lead approves in the PR that adds the file before Stage 2 starts.
- Keep it short — one screen. This is not a replacement for `docs/requirements/*`, which stays the durable source of product scope; `intent.md` is the per-change trigger.

## Stage 2 — Design: fold requirements + design into one pass

- Claude reads the new `intent.md` plus the relevant existing docs (`docs/architecture/system-architecture.md`, `docs/domain-model.md`, the ADRs) and produces `docs/intent/NNNN-short-name.spec.md` alongside it: affected components (Care Graph / Navigation Graph / ServicePoint / Location Provider — keep ADR-0002 and ADR-0004's boundaries explicit), API contract changes, open questions.
- If the change crosses one of the architecture boundaries in the root `README.md` (e.g. touching HIS access directly, merging Care Graph and Navigation Graph), Claude must flag it and require a new or amended ADR before Stage 3 — this is the "policy as skill" check from the playbook, encoded here as an explicit rule rather than a formal skill file for now.
- If the change is big enough to be a real architectural decision, it graduates to a numbered ADR in `docs/adr/` instead of staying a one-off spec.

## Stage 3 — Build: plan before code, then implement

**Add `CLAUDE.md` at repo root** (done as part of this change — see below) so every session starts with the same institutional knowledge: layout, commands, architecture boundaries, conventions. Update it whenever Claude repeats a mistake.

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

CarePath currently has zero automated tests, so this is the highest-leverage gap to close before Stage 5 gating means anything.

- **Go services** (`apps/api`, `apps/mock-his`): add table-driven tests per handler/domain function; `go test ./...` becomes the loop Claude iterates against before you look at a diff.
- **Web** (`apps/web`): add Vitest + React Testing Library; component and routing-flow tests.
- **Contract tests**: validate that `apps/api` and `apps/mock-his` handlers actually satisfy `packages/contracts/openapi/*.yaml` — this is the concrete, automatable version of ADR-0006's intent.
- **Bug-fix pattern**: write the failing test first, have Claude confirm it fails for the right reason, then fix the code without touching the test.
- **Evals**: skip a formal eval harness for now (20-50 task suites are overkill pre-CI); revisit once there's a stable API surface to regress against — e.g. routing correctness for the Navigation Graph is a good future eval candidate.

## Stage 5 — Deploy: start with CI, add gates as the team grows

- **Add `.github/workflows/ci.yml`**: matrix job running `go vet`/`go test ./...` for both Go modules (`go.work` already ties them together) and `npm run build`/tests for `apps/web`, plus `gofmt -l` / lint checks. This is the single biggest missing piece — right now nothing verifies a PR before merge.
- **`REVIEW.md`**: define what every PR gets checked for — correctness, adherence to the architecture boundaries in the root README, PHI/security handling, and OpenAPI contract compatibility. Claude can run this pass on every PR; a human reviewer then focuses on intent and risk instead of re-deriving the checklist.
- **Deploy gating**: given `docker-compose.yml`/`infra/docker/*` is the current deploy target (local/demo), there's no production gate yet. When a real staging/prod split exists, apply the playbook's rule directly: autonomous deploys allowed to staging, a named human approval required for anything production-like.

## Stage 6 — Maintain: defer, but note the shape

Not worth building at hackathon/MVP scale — no production traffic, no on-call. When CarePath moves past MVP:
- Recurring security scans make sense early given patient-adjacent data, even before full control-band monitoring exists.
- A lightweight version of the loop: a failing CI run or a reported bug becomes a new `docs/intent/NNNN-*.md`, re-entering Stage 1 — this costs nothing to adopt now and is worth doing from day one even without automated triggers.

## Artifact chain (adapted)

```
docs/intent/NNNN-name.md        (Stage 1 — problem + why)
docs/intent/NNNN-name.spec.md   (Stage 2 — design, or promoted to docs/adr/ if architectural)
PR description "plan"           (Stage 3 — files, sequence, risks; plan-mode output)
PR diff + CI + REVIEW.md pass   (Stage 4/5 — tests prove it, review pass checks policy)
docs/intent/ (new entry)        (Stage 6 — incidents/bugs loop back to Stage 1)
```

`docs/adr/*` remains the durable "why" record for decisions that outlive a single feature, exactly as it's used today — this workflow doesn't replace it, just adds the lighter-weight per-feature layer around it.

## Adoption order

1. **Now**: `CLAUDE.md` (this change), start using Plan Mode for non-trivial work, start writing short `intent.md` files for new features.
2. **Next**: Go tests + Vitest setup, GitHub Actions CI (`ci.yml`), `REVIEW.md`.
3. **Later**: hooks for formatting/contract-protection, PHI-handling skill, contract tests.
4. **Post-MVP**: staging/prod deploy gating, security scan cadence, incident→intent loop automation.

Prioritize based on actual friction (per the playbook: "teams prioritize based on their unique bottlenecks rather than sequential rollout") — if requirements churn is the pain, invest in Stage 1/2 first; if broken merges are the pain, jump straight to Stage 5 CI.
