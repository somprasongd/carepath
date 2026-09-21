ALTER TABLE carepath.floor DROP CONSTRAINT IF EXISTS floor_active_plan_id_fkey;
ALTER TABLE carepath.floor DROP COLUMN IF EXISTS active_plan_id;
ALTER TABLE carepath.floor DROP COLUMN IF EXISTS viewbox;
DROP TABLE IF EXISTS carepath.floor_plan;
