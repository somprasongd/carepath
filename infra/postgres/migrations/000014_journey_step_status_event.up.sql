-- #85 / NFR-09: append-only timeline of every journey step status change.
-- journey_step holds only current state (UpsertVisit replaces every step on
-- each replan), and journey_command_audit records staff commands only, with
-- no from_status and no visibility into planner-driven changes — neither can
-- answer "how did this step's status travel over time", which FR-22's wait
-- times and NFR-09's full audit trail both need.
--
-- kind and service_point_id are deliberate copies at event time: a later
-- replan may withdraw the step from the plan entirely (ADR-0009 §7), and
-- reports must still be able to describe the withdrawn step.
--
-- The table is append-only — no application code updates or deletes rows;
-- ON DELETE CASCADE only tidies up when a visit itself is deleted. It grows
-- without a retention policy: acceptable at hackathon scale, revisit before
-- any real deployment.
CREATE TABLE IF NOT EXISTS carepath.journey_step_status_event (
    event_id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    visit_id         text NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    step_key         text NOT NULL,
    kind             text NOT NULL,
    service_point_id text,
    from_status      text,                   -- NULL = step just entered the plan
    to_status        text NOT NULL,
    source           text NOT NULL,          -- 'planner' | command source, e.g. 'staff-web'
    actor_user_id    text,
    actor_username   text,
    occurred_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS journey_step_status_event_visit_idx
    ON carepath.journey_step_status_event (visit_id, occurred_at);
CREATE INDEX IF NOT EXISTS journey_step_status_event_service_point_idx
    ON carepath.journey_step_status_event (service_point_id, occurred_at);
CREATE INDEX IF NOT EXISTS journey_step_status_event_to_status_idx
    ON carepath.journey_step_status_event (to_status, occurred_at);
