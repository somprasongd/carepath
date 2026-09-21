-- #105 / ADR-0015: floor plans become uploaded artifacts instead of files in
-- the repository, because FR-11 needs plans an admin can replace and
-- apps/web can no longer bundle them from packages/floorplans.
--
-- Rows are append-only: a plan is never updated or deleted, floor.active_plan_id
-- points at the live one, and a rollback is a pointer move. That, plus
-- created_by, makes this table its own upload audit.
--
-- svg is what internal/floorplan serialized and what gets served; svg_raw is
-- the file as uploaded, kept so an admin can download back what they sent and
-- never sent to a patient.

-- The floor's coordinate space. Every place.x/y (000007) and nav_node.x/y
-- (000008) is a point in it, so a plan that declares a different one would
-- move every pin and route with no error anywhere: the first plan for a floor
-- sets this, and later uploads are held to it.
ALTER TABLE carepath.floor ADD COLUMN IF NOT EXISTS viewbox TEXT;

CREATE TABLE IF NOT EXISTS carepath.floor_plan (
    plan_id    TEXT PRIMARY KEY,
    floor_id   TEXT NOT NULL REFERENCES carepath.floor (floor_id),
    sha256     TEXT NOT NULL,              -- of svg; also the cache key in the serving URL
    viewbox    TEXT NOT NULL,              -- always equal to floor.viewbox once set
    svg        TEXT NOT NULL,              -- normalized, trusted, served
    svg_raw    TEXT NOT NULL,              -- as uploaded, ADMIN download only
    warnings   JSONB NOT NULL DEFAULT '[]'::jsonb, -- what the plan and the map model disagree about
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- NULL for the plans seeded from packages/floorplans, which no admin uploaded.
    created_by TEXT REFERENCES carepath.app_user (user_id),
    UNIQUE (floor_id, sha256)
);

CREATE INDEX IF NOT EXISTS floor_plan_floor_idx ON carepath.floor_plan (floor_id, created_at DESC);

ALTER TABLE carepath.floor ADD COLUMN IF NOT EXISTS active_plan_id TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'floor_active_plan_id_fkey'
    ) THEN
        ALTER TABLE carepath.floor
            ADD CONSTRAINT floor_active_plan_id_fkey
            FOREIGN KEY (active_plan_id) REFERENCES carepath.floor_plan (plan_id);
    END IF;
END $$;
