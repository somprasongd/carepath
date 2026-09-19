-- #19: minimal audit of transition commands CarePath forwards to the HIS.
-- Operational metadata only (timestamp, source, idempotency key) — step
-- status itself stays HIS-owned per ADR-0008.
CREATE TABLE IF NOT EXISTS carepath.journey_command_audit (
    command_id TEXT PRIMARY KEY,
    visit_id   TEXT NOT NULL,
    sequence   INT  NOT NULL,
    to_status  TEXT NOT NULL,
    source     TEXT NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
