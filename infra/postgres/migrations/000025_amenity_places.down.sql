DELETE FROM carepath.place
WHERE place_id IN ('RESTROOM-01', 'FOOD-01', 'RESTROOM-02');

ALTER TABLE carepath.place DROP CONSTRAINT IF EXISTS place_place_type_check;
ALTER TABLE carepath.place ADD CONSTRAINT place_place_type_check CHECK (place_type IN
    ('ROOM', 'COUNTER', 'WAITING_AREA', 'RESTROOM', 'ELEVATOR', 'STAIRWAY', 'RAMP', 'ENTRANCE', 'AMENITY'));
