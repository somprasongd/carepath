-- #26 down: unwind the navigation graph. Reverse order of the up: release
-- the place FK, restore floor-local entry node ids, drop the places this
-- migration seeded, then the graph tables themselves.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'place_entry_node_id_fkey'
    ) THEN
        ALTER TABLE carepath.place DROP CONSTRAINT place_entry_node_id_fkey;
    END IF;
END $$;

UPDATE carepath.place
SET entry_node_id = split_part(entry_node_id, '/', 2)
WHERE entry_node_id LIKE '%/%';

DELETE FROM carepath.place WHERE place_id IN (
    'CASHIER-01', 'WAITING-01', 'CT-01', 'TRIAGE-01', 'ER-EXAM-04', 'WOUND-01',
    'WELLNESS-NS-01', 'PRECOUNSEL-01', 'BOF-01', 'LONGEVITY-01'
);

DROP TABLE IF EXISTS carepath.nav_edge;
DROP TABLE IF EXISTS carepath.nav_node;
