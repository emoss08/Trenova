CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE EXTENSION IF NOT EXISTS unaccent;

CREATE TYPE gender_enum AS ENUM(
    'Male',
    'Female',
    'Other'
);

