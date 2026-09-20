-- #88 demo history: remove only what the up migration fabricated. The
-- VISIT-H-% prefix is the ownership mark — deleting the visits cascades to
-- their journey_step and journey_step_status_event rows; no service_point
-- or place rows were added, and real visits (ingest-created) are untouched.
DELETE FROM carepath.journey_visit WHERE visit_id LIKE 'VISIT-H-%';
