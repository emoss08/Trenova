CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE EXTENSION IF NOT EXISTS unaccent;

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TYPE gender_enum AS ENUM(
    'Male',
    'Female',
    'Other'
);

