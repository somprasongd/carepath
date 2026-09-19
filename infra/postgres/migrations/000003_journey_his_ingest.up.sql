-- Journey projection of canonical HIS events (#21, ADR-0008): CarePath-owned
-- derived state only; the HIS remains the system of record for visit and step
-- status.
CREATE TABLE IF NOT EXISTS carepath.journey_visit (
    visit_id    text PRIMARY KEY,
    patient_ref text NOT NULL,
    status      text NOT NULL,
    synced_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS carepath.journey_step (
    visit_id         text NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    sequence         integer NOT NULL,
    service_code     text NOT NULL,
    status           text NOT NULL,
    -- NULL = no service point configured for the external service code
    service_point_id text,
    PRIMARY KEY (visit_id, sequence)
);

-- Consumer-side idempotency: eventId is the dedupe key (ADR-0008 §3).
CREATE TABLE IF NOT EXISTS carepath.his_applied_event (
    event_id    text PRIMARY KEY,
    visit_id    text NOT NULL,
    applied_at  timestamptz NOT NULL DEFAULT now()
);

-- Durable feed cursor of the HIS ingest poller (single row).
CREATE TABLE IF NOT EXISTS carepath.his_ingest_state (
    singleton      boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    last_event_id  text NOT NULL DEFAULT '',
    updated_at     timestamptz NOT NULL DEFAULT now()
);

INSERT INTO carepath.his_ingest_state (singleton) VALUES (true) ON CONFLICT DO NOTHING;
