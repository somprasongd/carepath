-- Mock HIS's own schema — owned and written only by apps/mock-his.
-- CarePath (apps/api) must never query these tables directly (ADR-0005, ADR-0007);
-- it only ever calls the Mock HIS HTTP API through the HIS adapter.
--
-- For the hackathon this schema lives in the same Postgres instance as
-- `carepath` (one docker-compose postgres service, two schemas) so mock-his
-- doesn't need its own database container. apps/mock-his would connect with
-- its own DATABASE_URL / search_path, never sharing a connection or a
-- transaction with apps/api. In a production deployment the real HIS is a
-- genuinely separate system, so this schema split is also what a real HIS
-- adapter migration would replace.

CREATE SCHEMA IF NOT EXISTS his;

-- Demo/mock identity only — never real patient data (brief §12).
CREATE TABLE IF NOT EXISTS his.patient (
    patient_id  text PRIMARY KEY,
    mrn         text NOT NULL UNIQUE,      -- opaque mock hospital number, not a real HN
    full_name   text NOT NULL,             -- mock data only
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS his.visit (
    visit_id    text PRIMARY KEY,
    patient_id  text NOT NULL REFERENCES his.patient (patient_id),
    status      text NOT NULL CHECK (status IN ('ACTIVE', 'COMPLETED', 'CANCELLED')),
    opened_at   timestamptz NOT NULL DEFAULT now(),
    closed_at   timestamptz
);

CREATE INDEX IF NOT EXISTS visit_patient_id_idx ON his.visit (patient_id);

-- One row per concrete step of one visit. `display_order` is free to renumber;
-- prerequisite ("must happen after") relationships live in visit_step_dependency,
-- not in the ordering column, so inserting an unplanned step never has to
-- renumber or break an existing before/after condition.
CREATE TABLE IF NOT EXISTS his.visit_step (
    visit_step_id   text PRIMARY KEY,
    visit_id        text NOT NULL REFERENCES his.visit (visit_id),
    service_code    text NOT NULL,          -- matches carepath.service_point.code
    display_order   integer NOT NULL,
    status          text NOT NULL CHECK (status IN ('PENDING', 'READY', 'STARTED', 'COMPLETED', 'CANCELLED')),
    is_planned      boolean NOT NULL DEFAULT true, -- false = inserted mid-visit (M7 unplanned step)
    due_by          timestamptz,             -- optional deadline, e.g. "must draw blood before 11:00"
    arrived_at      timestamptz,
    started_at      timestamptz,
    completed_at    timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS visit_step_visit_id_idx ON his.visit_step (visit_id);
CREATE UNIQUE INDEX IF NOT EXISTS visit_step_visit_order_key ON his.visit_step (visit_id, display_order);

-- Self-referencing many-to-many: "this step requires that step to be COMPLETED first".
CREATE TABLE IF NOT EXISTS his.visit_step_dependency (
    visit_step_id           text NOT NULL REFERENCES his.visit_step (visit_step_id),
    requires_visit_step_id  text NOT NULL REFERENCES his.visit_step (visit_step_id),
    PRIMARY KEY (visit_step_id, requires_visit_step_id),
    CHECK (visit_step_id <> requires_visit_step_id)
);
