-- CarePath's own schema — owned and written only by apps/api.
-- Columns that reference a Mock HIS record (visit_id, visit_step_id) are opaque
-- text values, not database-level foreign keys: the his schema (see
-- mock-his-schema.sql) is owned by a different service and must never be
-- queried directly by apps/api (ADR-0005), so referential integrity across
-- that boundary is enforced by the HIS adapter/service layer, the same way
-- ADR-0007 already accepts convention-enforced boundaries between modules.
--
-- This file supersedes/extends infra/postgres/migrations/000001_init_schema.up.sql
-- (which already created carepath.service_point). Once each module (hospitalmap,
-- pathway, queue, identity) is actually implemented, split the matching section
-- below into its own infra/postgres/migrations/NNN_*.up.sql / .down.sql pair.

CREATE SCHEMA IF NOT EXISTS carepath;

-- ── Hospital map (M1) ───────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS carepath.building (
    building_id text PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    name        text NOT NULL
);

CREATE TABLE IF NOT EXISTS carepath.floor (
    floor_id     text PRIMARY KEY,
    building_id  text NOT NULL REFERENCES carepath.building (building_id),
    code         text NOT NULL,           -- e.g. "1", "2", "G"
    name         text NOT NULL,
    level_order  integer NOT NULL,        -- vertical display/sort order
    UNIQUE (building_id, code)
);

CREATE TABLE IF NOT EXISTS carepath.zone (
    zone_id  text PRIMARY KEY,
    floor_id text NOT NULL REFERENCES carepath.floor (floor_id),
    code     text NOT NULL,
    name     text NOT NULL,
    UNIQUE (floor_id, code)
);

CREATE TABLE IF NOT EXISTS carepath.place (
    place_id   text PRIMARY KEY,
    floor_id   text NOT NULL REFERENCES carepath.floor (floor_id),
    zone_id    text REFERENCES carepath.zone (zone_id),
    code       text NOT NULL UNIQUE,
    name       text NOT NULL,
    place_type text NOT NULL CHECK (place_type IN
        ('ROOM', 'COUNTER', 'WAITING_AREA', 'RESTROOM', 'ELEVATOR', 'STAIRWAY', 'RAMP', 'ENTRANCE', 'AMENITY'))
);

-- Walkable graph used for routing. Every connection is a directed row:
-- a normal two-way corridor is two rows (A→B and B→A); a one-way fire
-- escape stair is a single row. An elevator that only stops on some floors
-- simply has no edge row to the nodes on the floors it skips — no extra
-- "floor restriction" column needed.
CREATE TABLE IF NOT EXISTS carepath.nav_node (
    node_id   text PRIMARY KEY,
    floor_id  text NOT NULL REFERENCES carepath.floor (floor_id),
    place_id  text REFERENCES carepath.place (place_id), -- null for a bare corridor junction
    x         numeric NOT NULL,
    y         numeric NOT NULL,
    node_type text NOT NULL CHECK (node_type IN
        ('ROOM', 'JUNCTION', 'ELEVATOR', 'STAIR', 'RAMP', 'ENTRANCE'))
);

CREATE TABLE IF NOT EXISTS carepath.nav_edge (
    edge_id          text PRIMARY KEY,
    from_node_id     text NOT NULL REFERENCES carepath.nav_node (node_id),
    to_node_id       text NOT NULL REFERENCES carepath.nav_node (node_id),
    distance_m       numeric NOT NULL,
    walk_time_sec    integer NOT NULL,
    connection_type  text NOT NULL CHECK (connection_type IN ('CORRIDOR', 'ELEVATOR', 'STAIRWAY', 'RAMP')),
    wheelchair_ok    boolean NOT NULL DEFAULT true, -- false for stairs; lets routing avoid them
    CHECK (from_node_id <> to_node_id)
);

CREATE INDEX IF NOT EXISTS nav_edge_from_node_idx ON carepath.nav_edge (from_node_id);
CREATE INDEX IF NOT EXISTS nav_edge_to_node_idx ON carepath.nav_edge (to_node_id);

-- ── Service point (existing table; extend with the FK that place now allows) ─

-- Already created by infra/postgres/migrations/000001_init_schema.up.sql:
--   carepath.service_point (id, code, name, place_id, active)
-- Once carepath.place is migrated, add:
-- ALTER TABLE carepath.service_point
--     ADD CONSTRAINT service_point_place_id_fkey FOREIGN KEY (place_id) REFERENCES carepath.place (place_id);

CREATE TABLE IF NOT EXISTS carepath.service_point_hours (
    service_point_id text NOT NULL REFERENCES carepath.service_point (id),
    day_of_week      smallint NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0 = Sunday
    opens_at         time NOT NULL,
    closes_at        time NOT NULL,
    PRIMARY KEY (service_point_id, day_of_week)
);

-- ── Identity & RBAC (M8) ─────────────────────────────────────────────────────
-- Defined before pathway templates below, since pathway_template.created_by
-- references app_user.

CREATE TABLE IF NOT EXISTS carepath.app_user (
    user_id       text PRIMARY KEY,
    username      text NOT NULL UNIQUE,
    password_hash text NOT NULL, -- argon2id PHC string (ADR-0010); never plaintext (NFR-08)
                                 -- $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
    full_name     text NOT NULL,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS carepath.role (
    role_id text PRIMARY KEY,
    code    text NOT NULL UNIQUE CHECK (code IN
        ('PATIENT', 'STAFF', 'REGISTRATION_STAFF', 'SERVICE_POINT_STAFF', 'ADMIN', 'EXECUTIVE')),
    name    text NOT NULL
);

CREATE TABLE IF NOT EXISTS carepath.user_role (
    user_id text NOT NULL REFERENCES carepath.app_user (user_id),
    role_id text NOT NULL REFERENCES carepath.role (role_id),
    PRIMARY KEY (user_id, role_id)
);

-- Staff refresh tokens (ADR-0010 §4). The access token is a short-lived JWT and
-- is deliberately NOT stored; only the long-lived, revocable half is a row. The
-- token itself is never persisted — token_hash is sha256(token), so a dump of
-- this table yields nothing a thief can present.
CREATE TABLE IF NOT EXISTS carepath.refresh_token (
    token_hash text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES carepath.app_user (user_id) ON DELETE CASCADE,
    issued_at  timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz, -- non-null = spent by a rotation; replaying it revokes the user's set
    revoked_at timestamptz  -- non-null = logged out, or revoked by reuse detection
);

CREATE INDEX IF NOT EXISTS refresh_token_user_idx ON carepath.refresh_token (user_id);
CREATE INDEX IF NOT EXISTS refresh_token_expires_at_idx ON carepath.refresh_token (expires_at);

-- Every status change on a visit/step/queue ticket, regardless of which table
-- it touched (NFR-09). old_value/new_value are minimal JSON snapshots, not full
-- row dumps, to avoid duplicating sensitive fields into an append-only log.
CREATE TABLE IF NOT EXISTS carepath.audit_log (
    audit_log_id bigserial PRIMARY KEY,
    user_id      text REFERENCES carepath.app_user (user_id), -- null = system-initiated
    entity_type  text NOT NULL,   -- e.g. 'visit_step', 'queue_ticket'
    entity_id    text NOT NULL,
    action       text NOT NULL,   -- e.g. 'STATUS_CHANGED'
    old_value    jsonb,
    new_value    jsonb,
    changed_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_log_entity_idx ON carepath.audit_log (entity_type, entity_id);

-- ── Care pathway templates (M2) ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS carepath.care_category (
    care_category_id text PRIMARY KEY,
    code             text NOT NULL UNIQUE,
    name             text NOT NULL
);

CREATE TABLE IF NOT EXISTS carepath.pathway_template (
    pathway_template_id text PRIMARY KEY,
    care_category_id    text NOT NULL REFERENCES carepath.care_category (care_category_id),
    code                 text NOT NULL,
    name                 text NOT NULL,
    version              integer NOT NULL DEFAULT 1,
    is_active            boolean NOT NULL DEFAULT true,
    created_by           text REFERENCES carepath.app_user (user_id),
    created_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (code, version)
);

CREATE TABLE IF NOT EXISTS carepath.pathway_template_step (
    pathway_template_step_id text PRIMARY KEY,
    pathway_template_id      text NOT NULL REFERENCES carepath.pathway_template (pathway_template_id),
    step_order               integer NOT NULL,
    service_code             text NOT NULL, -- matches carepath.service_point.code
    name                     text NOT NULL,
    is_optional              boolean NOT NULL DEFAULT false,
    UNIQUE (pathway_template_id, step_order)
);

-- Prerequisite graph at the template level; copied forward into
-- his.visit_step_dependency when a visit is registered from this template.
CREATE TABLE IF NOT EXISTS carepath.pathway_template_step_dependency (
    pathway_template_step_id          text NOT NULL REFERENCES carepath.pathway_template_step (pathway_template_step_id),
    requires_pathway_template_step_id text NOT NULL REFERENCES carepath.pathway_template_step (pathway_template_step_id),
    PRIMARY KEY (pathway_template_step_id, requires_pathway_template_step_id),
    CHECK (pathway_template_step_id <> requires_pathway_template_step_id)
);

-- ── Queue (M7, S1) ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS carepath.queue_ticket (
    queue_ticket_id  text PRIMARY KEY,
    service_point_id text NOT NULL REFERENCES carepath.service_point (id),
    visit_step_id    text NOT NULL, -- opaque reference to his.visit_step (a different service's schema)
    ticket_number    integer NOT NULL,
    status           text NOT NULL CHECK (status IN ('WAITING', 'CALLED', 'SERVING', 'DONE', 'SKIPPED')),
    created_at       timestamptz NOT NULL DEFAULT now(),
    called_at        timestamptz
);

CREATE INDEX IF NOT EXISTS queue_ticket_service_point_status_idx
    ON carepath.queue_ticket (service_point_id, status);

-- ── Location (S2, optional Zigbee) ──────────────────────────────────────────

CREATE TABLE IF NOT EXISTS carepath.tag_assignment (
    tag_assignment_id text PRIMARY KEY,
    visit_id          text NOT NULL, -- opaque reference to his.visit (a different service's schema)
    tag_code          text NOT NULL,
    assigned_at       timestamptz NOT NULL DEFAULT now(),
    released_at       timestamptz
);

CREATE TABLE IF NOT EXISTS carepath.location_observation (
    location_observation_id bigserial PRIMARY KEY,
    tag_assignment_id       text REFERENCES carepath.tag_assignment (tag_assignment_id),
    visit_id                text NOT NULL, -- opaque reference to his.visit; present even for QR/manual (no tag)
    place_id                text REFERENCES carepath.place (place_id),
    source                  text NOT NULL CHECK (source IN ('QR', 'ZIGBEE', 'MANUAL')),
    confidence              numeric NOT NULL DEFAULT 1.0,
    observed_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS location_observation_visit_id_idx ON carepath.location_observation (visit_id);

-- ── Relative tracking link (C2, stretch) ────────────────────────────────────

CREATE TABLE IF NOT EXISTS carepath.visit_share_link (
    token      text PRIMARY KEY, -- opaque, unguessable
    visit_id   text NOT NULL,    -- opaque reference to his.visit (a different service's schema)
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- ── Reference data ───────────────────────────────────────────────────────────

-- The full role catalogue this design allows. Per ADR-0010 the MVP seeds and
-- enforces only ADMIN and STAFF (see the STAFF row below); the rest stay here
-- as the growth path — adding one later is an insert, not a migration.
INSERT INTO carepath.role (role_id, code, name) VALUES
    ('ROLE-PATIENT', 'PATIENT', 'Patient'),
    ('ROLE-STAFF', 'STAFF', 'Hospital Staff'),
    ('ROLE-REGISTRATION', 'REGISTRATION_STAFF', 'Registration / Screening Staff'),
    ('ROLE-SERVICE-POINT', 'SERVICE_POINT_STAFF', 'Service-Point Staff'),
    ('ROLE-ADMIN', 'ADMIN', 'Hospital Admin'),
    ('ROLE-EXECUTIVE', 'EXECUTIVE', 'Hospital Executive')
ON CONFLICT (role_id) DO NOTHING;

-- Demo accounts (ADR-0010 §9): admin/demo and staff/demo. The hashes below are
-- placeholders in this design script — the real migration commits the actual
-- argon2id encoding of "demo", which makes these credentials public by
-- construction: local development and the hackathon demo only. Before any
-- deployment others can reach, delete or deactivate both rows (ADR-0010 §12).
INSERT INTO carepath.app_user (user_id, username, password_hash, full_name) VALUES
    ('USER-ADMIN', 'admin', '$argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>', 'ผู้ดูแลระบบ (demo)'),
    ('USER-STAFF', 'staff', '$argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>', 'เจ้าหน้าที่ (demo)')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO carepath.user_role (user_id, role_id) VALUES
    ('USER-ADMIN', 'ROLE-ADMIN'),
    ('USER-STAFF', 'ROLE-STAFF')
ON CONFLICT DO NOTHING;
