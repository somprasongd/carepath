# 2. ER Diagram and Database Structure (แผนภาพ ER และโครงสร้างฐานข้อมูล)

**Deliverable 2 of 6** per the hackathon brief §10: an ER diagram, SQL scripts to create the tables, and an explanation of normalization. This design covers the full requirement scope from [deliverable 1](01-requirement-specification.md) (Must/Should/Could), following the brief's own instruction (§7) that the team designs the ER diagram itself — the brief's entity list is a starting point, not a checklist.

## 2.1 What's already real vs. what this document adds

Today the only table actually migrated is `carepath.service_point` (`infra/postgres/migrations/000001_init_schema.up.sql`), which is why the tool-generated snapshot at [`docs/architecture/erd/`](../architecture/erd/) shows just that one table — it's accurate, not stale, for what exists right now. Mock HIS (`apps/mock-his`) has no database at all yet; it serves one hardcoded visit from an in-memory map.

This document designs the full schema needed for the Must/Should/Could scope, as two runnable SQL scripts:

- [`sql/mock-his-schema.sql`](sql/mock-his-schema.sql) — the `his` schema (owned by `apps/mock-his`)
- [`sql/carepath-schema.sql`](sql/carepath-schema.sql) — the `carepath` schema (owned by `apps/api`), extending the existing `service_point` table

**Both scripts were verified end-to-end**: applied in order (`000001_init_schema.up.sql` → `000002_seed_service_points.up.sql` → `mock-his-schema.sql` → `carepath-schema.sql`) against a disposable PostgreSQL 17 container (matching the version pinned in `docker-compose.yml`), then inspected with `tbls` (the same tool `make docs-erd` uses) to confirm every foreign key resolves and no table is orphaned. When these tables are actually implemented module-by-module, split each section into its own `infra/postgres/migrations/NNN_*.up.sql`/`.down.sql` pair and re-run `make docs-erd` to regenerate `docs/architecture/erd/` for real.

## 2.2 One database, two schemas — and why

ADR-0005 requires that "CarePath must not query the HIS database directly." For the hackathon, both schemas run in the single `postgres:17-alpine` container already in `docker-compose.yml` (no second database container to stand up under time pressure), but the boundary is kept as sharp as if they were physically separate:

| | `his` schema | `carepath` schema |
|---|---|---|
| Owned/written by | `apps/mock-his` only | `apps/api` only |
| Holds | Patient, Visit, clinic assignments, orders, encounters — the clinical facts a real hospital HIS would own (no step list; ADR-0009) | Hospital map, service points, the derived journey plan (`journey_step`), queue, users/roles, audit log, location — CarePath's own operational and spatial data |
| Cross-references | none into `carepath` | Stores HIS ids (`visit_id`, `order_id`) as **opaque text, not a database foreign key** — `apps/api` reaches HIS data only through the HIS adapter/API (ADR-0005, ADR-0009), never a cross-schema SQL join |

This mirrors the trade-off ADR-0007 already accepts for cross-module boundaries inside `apps/api` ("module discipline relies on review rather than the compiler") — here it's enforced by which service holds which `DATABASE_URL`, not by a database-level constraint. If/when a real HIS replaces Mock HIS, only the adapter changes; nothing in the `carepath` schema does.

## 2.3 ER diagrams

Grouped the same way the brief itself groups the data (§7), plus RBAC/audit for M8/NFR-09.

### A. Hospital map (M1)

```mermaid
erDiagram
    BUILDING ||--o{ FLOOR : contains
    FLOOR ||--o{ ZONE : contains
    FLOOR ||--o{ PLACE : contains
    ZONE ||--o{ PLACE : groups
    FLOOR ||--o{ NAV_NODE : contains
    PLACE ||--o| NAV_NODE : "located at"
    NAV_NODE ||--o{ NAV_EDGE : "from"
    NAV_NODE ||--o{ NAV_EDGE : "to"
    PLACE ||--o| SERVICE_POINT : hosts
    SERVICE_POINT ||--o{ SERVICE_POINT_HOURS : "open hours"

    BUILDING {
        text building_id PK
        text code
        text name
    }
    FLOOR {
        text floor_id PK
        text building_id FK
        text code
        text name
        int level_order
    }
    ZONE {
        text zone_id PK
        text floor_id FK
        text code
        text name
    }
    PLACE {
        text place_id PK
        text floor_id FK
        text zone_id FK
        text code
        text name
        text place_type
    }
    NAV_NODE {
        text node_id PK
        text floor_id FK
        text place_id FK
        numeric x
        numeric y
        text node_type
    }
    NAV_EDGE {
        text edge_id PK
        text from_node_id FK
        text to_node_id FK
        numeric distance_m
        int walk_time_sec
        text connection_type
        boolean wheelchair_ok
    }
    SERVICE_POINT {
        text id PK
        text code
        text name
        text place_id FK
        boolean active
    }
    SERVICE_POINT_HOURS {
        text service_point_id FK
        smallint day_of_week
        time opens_at
        time closes_at
    }
```

### B. Patient and visit (`his` schema) — revised per ADR-0009

> **Superseded design note.** The version of this section originally written here modeled `his.visit_step` with `display_order` and `visit_step_dependency` — i.e., it assumed the HIS reports (and CarePath must not overwrite) an ordered step list. [ADR-0009](../adr/0009-carepath-owns-journey-plan.md) found that assumption doesn't hold: the HIS has no step concept at all. It only knows a visit, which clinic(s) it's assigned to, and orders (with a performed/resulted lifecycle) and encounter-completion facts. The diagram below replaces the old `VISIT_STEP`/`VISIT_STEP_DEPENDENCY` pair with what the HIS actually has; the derived, ordered plan now lives in `carepath.journey_step` (§B′ below), which is CarePath-owned, not `his`-owned.

```mermaid
erDiagram
    PATIENT ||--o{ VISIT : has
    VISIT ||--o{ VISIT_CLINIC : "assigned to"
    VISIT ||--o{ ORDER_ : places
    VISIT ||--o{ ENCOUNTER : "seen at"

    PATIENT {
        text patient_id PK
        text mrn
        text full_name
        timestamptz created_at
    }
    VISIT {
        text visit_id PK
        text patient_id FK
        text visit_type "WALKIN or APPOINTMENT"
        text status
        timestamptz opened_at
        timestamptz closed_at
    }
    VISIT_CLINIC {
        text visit_id FK
        text clinic_code
        int assignment_order
    }
    ORDER_ {
        text order_id PK
        text visit_id FK
        text order_type "LAB, XRAY, EKG, US, DRUG"
        text order_name
        text ordered_by_clinic
        timestamptz ordered_at
        text status "PLACED, PERFORMED, RESULTED, CANCELLED"
        timestamptz performed_at
        timestamptz resulted_at
    }
    ENCOUNTER {
        text visit_id FK
        text clinic_code
        int round
        timestamptz completed_at "set once the clinic confirms this round is done"
    }
```

*(`ORDER_` is named with a trailing underscore only because `ORDER` is a reserved SQL keyword; the real column/table name is `order`, quoted.)*

### B′. Journey plan (`carepath` schema, ADR-0009) — replaces the old Care Pathway Template step list

The plan CarePath derives from the facts in §B, addressed by a stable `step_key` rather than a renumberable sequence — see [`internal/journey/planner.go`](../../apps/api/internal/journey/planner.go) for the derivation rules and [`infra/postgres/migrations/000010_journey_plan.up.sql`](../../infra/postgres/migrations/000010_journey_plan.up.sql) for the actual (already-implemented) table.

```mermaid
erDiagram
    JOURNEY_VISIT ||--o{ JOURNEY_STEP : contains
    JOURNEY_VISIT ||--o{ JOURNEY_CLOSED_ROUND : "round confirmed done"
    JOURNEY_STEP ||--o| SERVICE_POINT : "resolves to"

    JOURNEY_VISIT {
        text visit_id PK "HIS visit_id, opaque"
        text patient_ref
        text patient_name "the one PHI field CarePath stores, ADR-0009 §8"
        text status
        timestamptz synced_at
    }
    JOURNEY_STEP {
        text visit_id FK
        text step_key PK "e.g. CLINIC:MED:2, LAB:1 — stable across replans"
        int sequence "display order only, not identity"
        text kind "REGISTRATION, CLINIC, LAB, XRAY, EKG, ULTRASOUND, CASHIER, PHARMACY"
        text clinic_code
        int round
        text_array order_refs "HIS order ids this step represents"
        text status "PENDING, WAITING, READY, STARTED, COMPLETED, CANCELLED"
        text service_point_id FK
    }
    JOURNEY_CLOSED_ROUND {
        text visit_id FK
        text step_key "the CLINIC round confirmed finished"
        timestamptz closed_at
    }
```

### C. Care pathway templates (M2) — superseded by ADR-0009, kept for historical context

> **Do not build this table set.** [ADR-0009](../adr/0009-carepath-owns-journey-plan.md) found that "a per-patient template staff assemble ahead of time" doesn't fit how the HIS actually reports facts — the plan has to react to orders and encounters as they happen, not follow a pre-picked template. What this section called "pathway rules" now lives as ordering logic in [`internal/journey/planner.go`](../../apps/api/internal/journey/planner.go) (phase-based: registration → pre-visit diagnostics → clinic → mid-visit diagnostics → return-to-clinic → cashier → pharmacy), not as configuration rows. Kept below only so the design rationale for the original brief-driven M2 scope isn't lost.

```mermaid
erDiagram
    CARE_CATEGORY ||--o{ PATHWAY_TEMPLATE : classifies
    PATHWAY_TEMPLATE ||--o{ PATHWAY_TEMPLATE_STEP : contains
    PATHWAY_TEMPLATE_STEP ||--o{ PATHWAY_TEMPLATE_STEP_DEPENDENCY : "as step"
    PATHWAY_TEMPLATE_STEP ||--o{ PATHWAY_TEMPLATE_STEP_DEPENDENCY : "as prerequisite"
    APP_USER ||--o{ PATHWAY_TEMPLATE : authors

    CARE_CATEGORY {
        text care_category_id PK
        text code
        text name
    }
    PATHWAY_TEMPLATE {
        text pathway_template_id PK
        text care_category_id FK
        text code
        text name
        int version
        boolean is_active
        text created_by FK
    }
    PATHWAY_TEMPLATE_STEP {
        text pathway_template_step_id PK
        text pathway_template_id FK
        int step_order
        text service_code
        text name
        boolean is_optional
    }
    PATHWAY_TEMPLATE_STEP_DEPENDENCY {
        text pathway_template_step_id FK
        text requires_pathway_template_step_id FK
    }
```

### D. Queue and location (M7, S1, S2, C2)

```mermaid
erDiagram
    SERVICE_POINT ||--o{ QUEUE_TICKET : serves
    TAG_ASSIGNMENT ||--o{ LOCATION_OBSERVATION : produces
    PLACE ||--o{ LOCATION_OBSERVATION : "observed at"

    QUEUE_TICKET {
        text queue_ticket_id PK
        text service_point_id FK
        text visit_step_id "his.visit_step, app-level ref"
        int ticket_number
        text status
    }
    TAG_ASSIGNMENT {
        text tag_assignment_id PK
        text visit_id "his.visit, app-level ref"
        text tag_code
    }
    LOCATION_OBSERVATION {
        bigint location_observation_id PK
        text tag_assignment_id FK
        text visit_id "his.visit, app-level ref"
        text place_id FK
        text source
        numeric confidence
    }
    VISIT_SHARE_LINK {
        text token PK
        text visit_id "his.visit, app-level ref"
        timestamptz expires_at
    }
```

### E. Identity, RBAC, and audit (M8, NFR-09)

```mermaid
erDiagram
    APP_USER ||--o{ USER_ROLE : has
    ROLE ||--o{ USER_ROLE : "granted to"
    APP_USER ||--o{ AUDIT_LOG : performs

    APP_USER {
        text user_id PK
        text username
        text password_hash
        text full_name
        boolean is_active
    }
    ROLE {
        text role_id PK
        text code
        text name
    }
    USER_ROLE {
        text user_id FK
        text role_id FK
    }
    AUDIT_LOG {
        bigint audit_log_id PK
        text user_id FK
        text entity_type
        text entity_id
        text action
        jsonb old_value
        jsonb new_value
        timestamptz changed_at
    }
```

### Cross-schema references (application-enforced, not a DB foreign key)

| `carepath` column | Refers to | Enforced by |
|---|---|---|
| `queue_ticket.visit_step_id` | `his.visit_step.visit_step_id` | HIS adapter validates the step exists/is READY before a ticket is created |
| `tag_assignment.visit_id` | `his.visit.visit_id` | HIS adapter validates the visit is ACTIVE |
| `location_observation.visit_id` | `his.visit.visit_id` | same |
| `visit_share_link.visit_id` | `his.visit.visit_id` | link creation checks the visit is ACTIVE; expiry is time-based, not FK-based |

## 2.4 Design questions the brief raises (§7)

**How are nodes and edges related, and how does that stay efficient for shortest-path?**
`nav_edge` is a plain directed adjacency list (`from_node_id`, `to_node_id`, `distance_m`, `walk_time_sec`), indexed on both columns. Dijkstra/A* read it as a standard weighted directed graph — no recursive CTE or extra join needed to expand a "connection" into a direction.

**Are walking connections always bidirectional?**
No, and the schema doesn't force an answer either way: every connection is **one directed row**. An ordinary two-way corridor is simply two rows (A→B and B→A) with the same distance/time. A one-way fire-escape stair is a single row. An elevator that skips a floor needs no special "floor restriction" flag — it just has no edge row connecting to that floor's elevator node, which falls straight out of the graph model.

**How does an unplanned step get inserted without breaking existing before/after ordering?** *(revised under ADR-0009 — see §B′)*
The original answer here (`visit_step_dependency` self-referencing edges) assumed CarePath persists explicit prerequisite rows staff insert against. ADR-0009 replaces that with a **pure function**: `journey.Plan(visit, prior, closedRounds) -> []Step` recomputes the whole plan from HIS facts on every new event, and the result is diffed against what's stored — never hand-edited. "Ordering" is a `phase` number baked into the planner's rules (registration=0, pre-visit diagnostics=10, clinic=20, mid-visit diagnostics=30, return-to-clinic=40, cashier=90, pharmacy=95), not a per-row dependency edge. An "unplanned" order placed mid-visit (M7/FR-16) is simply a new fact (`order.placed`) the next replan picks up automatically — inserting `journey_step` rows for it and, if it was ordered while a clinic encounter was open, inferring a return-to-clinic step; no staff action creates a dependency row. `journey_step`'s own primary key (`visit_id`, `step_key`) is what "does not disturb existing rows" now means: a step that is `STARTED`/`COMPLETED`/`CANCELLED` is never reordered or removed by a replan (ADR-0009 §7) — only a still-`PENDING`/`WAITING` step may be added, removed, or reordered.

**What data is sensitive and shouldn't appear on a public screen?**
`his.patient.full_name`, `his.care_category`-derived diagnosis/category names, and any free-text visit detail must never be joined into a public-facing view — e.g., the service-point queue-call screen (US-14) should display only `queue_ticket.ticket_number`, never the patient's name or care category. This is [NFR-03](../requirements/non-functional-requirements.md) applied at the schema/view level, not just the API layer.

## 2.5 Normalization (3NF, per the brief's technical requirement §8)

- **1NF** — every column is atomic. Steps are rows in `visit_step`, not an array/CSV column on `visit`; prerequisites are rows in a dependency table, not a comma-separated list of step ids.
- **2NF** — every non-key column depends on the *whole* primary key. The composite-key join tables (`visit_step_dependency`, `pathway_template_step_dependency`, `user_role`, `service_point_hours`) carry no descriptive column beyond the key itself, so partial dependency can't arise.
- **3NF** — no non-key column depends on another non-key column (no transitive dependency):
  - `visit_step` stores `service_code`, not the service point's name/place — those are looked up from `carepath.service_point` by code.
  - `floor`/`place` store only `building_id`/`floor_id`, never a copied-down building or floor name.
  - `pathway_template_step` stores `service_code`, not a duplicated service-point name.
  - `user_role` stores no role name — it's looked up from `role` by `role_id`.

## 2.6 SQL scripts

- [`sql/mock-his-schema.sql`](sql/mock-his-schema.sql) — `his` schema: `patient`, `visit`, `visit_step`, `visit_step_dependency`
- [`sql/carepath-schema.sql`](sql/carepath-schema.sql) — `carepath` schema: hospital map, service-point hours, identity/RBAC, audit log, pathway templates, queue, location, relative-tracking links, plus role reference data

Apply order: `000001_init_schema.up.sql` → `000002_seed_service_points.up.sql` → `mock-his-schema.sql` → `carepath-schema.sql`.

## 2.7 Next step

These two scripts are a design deliverable, not yet wired into `golang-migrate`. When a module in the [technical blueprint](../architecture/technical-blueprint.md)'s "planned" list (`hospitalmap`, `journey`/pathway, `identity`) gets built, carve its tables out of these files into `infra/postgres/migrations/NNN_*.up.sql`/`.down.sql`, add the matching `.down.sql` (`DROP TABLE`/`DROP SCHEMA` in reverse order), and run `make docs-erd` so `docs/architecture/erd/` reflects the real, live schema again.
