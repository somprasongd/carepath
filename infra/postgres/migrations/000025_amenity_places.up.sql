-- #109 (FR-26): real amenity data for the map. The place_type vocabulary
-- already carried WAITING_AREA/RESTROOM/AMENITY since 000007 but nothing
-- used them — this adds FOOD_STALL so the three FR-26 categories are three
-- honest place kinds (a generic AMENITY cannot tell food from anything
-- else, and place type is the stable code the client catalogs by, ADR-0012).
--
-- Entries snap to existing corridor/lift nodes rather than adding dedicated
-- PLACE_ENTRY nodes: routing already reaches every node, and pinning new
-- graph nodes would drag packages/floorplans/graphs along for no MVP value.
-- x/y follow the 000007/000008 convention — the entry node's coordinates,
-- so the destination pin and the route endpoint are the same point.
-- WAITING-01 (000008) already covers the waiting-area category.

ALTER TABLE carepath.place DROP CONSTRAINT IF EXISTS place_place_type_check;
ALTER TABLE carepath.place ADD CONSTRAINT place_place_type_check CHECK (place_type IN
    ('ROOM', 'COUNTER', 'WAITING_AREA', 'RESTROOM', 'ELEVATOR', 'STAIRWAY', 'RAMP', 'ENTRANCE', 'AMENITY', 'FOOD_STALL'));

INSERT INTO carepath.place (place_id, floor_id, name, place_type, x, y, entry_node_id)
VALUES
    ('RESTROOM-01', 'I-1301', 'Restroom (Ground)', 'RESTROOM',   318, 190, 'I-1301/node-ramp'),
    ('FOOD-01',     'I-1301', 'Food Stall',        'FOOD_STALL', 620, 190, 'I-1301/node-waiting'),
    ('RESTROOM-02', 'I-1302', 'Restroom (Upper)',  'RESTROOM',   480, 295, 'I-1302/node-lift')
ON CONFLICT (place_id) DO NOTHING;
