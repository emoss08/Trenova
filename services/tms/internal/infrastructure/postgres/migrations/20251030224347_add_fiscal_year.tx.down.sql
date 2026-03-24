DROP TRIGGER IF EXISTS fiscal_years_search_update ON "fiscal_years";

--bun:split
DROP TRIGGER IF EXISTS trigger_enforce_single_current_fiscal_year ON "fiscal_years";

--bun:split
DROP FUNCTION IF EXISTS fiscal_years_search_trigger() CASCADE;

--bun:split
DROP FUNCTION IF EXISTS enforce_single_current_fiscal_year() CASCADE;

--bun:split
DROP TABLE IF EXISTS "fiscal_years" CASCADE;

--bun:split
DROP TYPE IF EXISTS fiscal_year_status_enum CASCADE;
