-- Undo only what this migration owns: a floor still pointed at a seeded
-- plan (never re-pointed by an admin), and the seeded plan rows themselves.
-- A floor an admin has since uploaded to is left alone — the up migration
-- is guarded the same way (AND active_plan_id IS NULL), so the two agree on
-- what "still on the seed" means. Without this scoping, a down/up cycle on
-- a database with an admin upload would null out every floor's pointer and
-- then silently reset it back to the seed, discarding the admin's plan.
UPDATE carepath.floor
SET active_plan_id = NULL, viewbox = NULL
WHERE active_plan_id IN (SELECT plan_id FROM carepath.floor_plan WHERE created_by IS NULL);

-- Safe now: the UPDATE above cleared every floor.active_plan_id that could
-- still reference one of these rows, so this can't hit the FK.
DELETE FROM carepath.floor_plan WHERE created_by IS NULL;
