-- FR-15 / #102: a staff user works at specific service point(s). This is
-- the assignment the staff queue console authorizes against — STAFF sees
-- the queue of a point only when assigned to it (ADMIN bypasses, matching
-- the RequireRole(STAFF, ADMIN) convention elsewhere). Reported as missing
-- on #102 before this migration: nothing linked users to points before.
--
-- Assignment is deliberately a plain many-to-many with no extra semantics:
-- the hackathon has one demo staff account per station concept, and richer
-- models (shifts, primary/backup) can evolve this table without a rename.
CREATE TABLE IF NOT EXISTS carepath.user_service_point (
    user_id          text NOT NULL REFERENCES carepath.app_user (user_id) ON DELETE CASCADE,
    service_point_id text NOT NULL REFERENCES carepath.service_point (id) ON DELETE CASCADE,
    assigned_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, service_point_id)
);

CREATE INDEX IF NOT EXISTS user_service_point_sp_idx ON carepath.user_service_point (service_point_id);

-- The demo staff account runs the laboratory draw station — the station the
-- queue console has pretended to be all along. ON CONFLICT DO NOTHING keeps
-- a re-run from fighting an operator's own inserts.
INSERT INTO carepath.user_service_point (user_id, service_point_id)
VALUES ('user-staff', 'SP-ORDERTYPE-LAB')
ON CONFLICT (user_id, service_point_id) DO NOTHING;
