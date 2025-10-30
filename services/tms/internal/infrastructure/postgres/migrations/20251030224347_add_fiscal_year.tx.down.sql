DROP TABLE IF EXISTS "fiscal_years";

DROP TYPE IF EXISTS fiscal_year_status_enum;

DROP TRIGGER IF EXISTS fiscal_years_search_update ON "fiscal_years";

DROP FUNCTION IF EXISTS fiscal_years_search_trigger();

DROP FUNCTION IF EXISTS enforce_single_current_fiscal_year();

DROP TRIGGER IF EXISTS trigger_enforce_single_current_fiscal_year ON fiscal_years;

