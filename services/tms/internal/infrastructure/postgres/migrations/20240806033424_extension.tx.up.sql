CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE EXTENSION IF NOT EXISTS pgstattuple;
CREATE EXTENSION IF NOT EXISTS pg_prewarm;

--bun:split
CREATE OR REPLACE FUNCTION immutable_to_tsvector(config regconfig, input text)
RETURNS tsvector
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$ SELECT to_tsvector(config, input) $$;

--bun:split
CREATE OR REPLACE FUNCTION enum_to_text(anyenum)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$ SELECT $1::text $$;

--bun:split
CREATE OR REPLACE FUNCTION immutable_array_to_string(anyarray, text)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE
AS $$ SELECT array_to_string($1, $2) $$;
