CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE gender_enum AS ENUM ('Male', 'Female', 'Other');