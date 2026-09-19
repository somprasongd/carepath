CREATE SCHEMA IF NOT EXISTS carepath;

CREATE TABLE IF NOT EXISTS carepath.service_point (
    id text PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    place_id text NOT NULL,
    active boolean NOT NULL DEFAULT true
);
