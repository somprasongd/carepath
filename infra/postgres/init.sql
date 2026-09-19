CREATE SCHEMA IF NOT EXISTS carepath;

CREATE TABLE IF NOT EXISTS carepath.service_point (
    id text PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    place_id text NOT NULL,
    active boolean NOT NULL DEFAULT true
);

INSERT INTO carepath.service_point (id, code, name, place_id)
VALUES
    ('SP-REG', 'REGISTRATION', 'Registration', 'REG-01'),
    ('SP-LAB', 'LAB', 'Laboratory', 'LAB-01'),
    ('SP-PHARMACY', 'PHARMACY', 'Pharmacy', 'PHARMACY-01')
ON CONFLICT (id) DO NOTHING;
