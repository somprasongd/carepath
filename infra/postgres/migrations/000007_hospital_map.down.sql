DELETE FROM carepath.service_point WHERE id = 'SP-OPD';

ALTER TABLE carepath.service_point DROP CONSTRAINT IF EXISTS service_point_place_id_fkey;

DROP TABLE IF EXISTS carepath.place;
DROP TABLE IF EXISTS carepath.floor;
DROP TABLE IF EXISTS carepath.building;
