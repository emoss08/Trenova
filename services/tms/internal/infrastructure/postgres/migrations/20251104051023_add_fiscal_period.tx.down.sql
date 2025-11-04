SET statement_timeout = 0;

--bun:split
DROP TRIGGER IF EXISTS fiscal_periods_search_update ON "fiscal_periods";

--bun:split
DROP FUNCTION IF EXISTS fiscal_periods_search_trigger() CASCADE;

--bun:split
DROP TABLE IF EXISTS "fiscal_periods" CASCADE;

--bun:split
DROP TYPE IF EXISTS period_status_enum CASCADE;

--bun:split
DROP TYPE IF EXISTS period_type_enum CASCADE;

