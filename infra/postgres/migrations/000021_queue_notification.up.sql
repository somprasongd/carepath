-- #104 (FR-21 / ADR-0013): queue-proximity notification state.
-- queue_notification is the exactly-once guard: one row per visit per step,
-- claimed with ON CONFLICT DO NOTHING before any send, so plan recomputation
-- (FR-28), queue wobble across the threshold, and API restarts cannot make
-- the same step notify twice. channel records what actually delivered.
CREATE TABLE IF NOT EXISTS carepath.queue_notification (
    visit_id  text        NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    step_key  text        NOT NULL,
    channel   text        NOT NULL DEFAULT 'noop',
    sent_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (visit_id, step_key)
);

-- visit_notification_pref is the patient's opt-out (#104): absence of a row
-- means enabled, so existing visits need no backfill and the default is
-- "notify me".
CREATE TABLE IF NOT EXISTS carepath.visit_notification_pref (
    visit_id   text        NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    enabled    boolean     NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (visit_id)
);
