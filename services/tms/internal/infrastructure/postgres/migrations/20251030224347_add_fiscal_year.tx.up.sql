CREATE TYPE fiscal_year_status_enum AS ENUM(
    'Draft',
    'Open',
    'Closed',
    'Locked'
);

CREATE TABLE IF NOT EXISTS "fiscal_years"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" fiscal_year_status_enum NOT NULL DEFAULT 'Draft',
    "year" int NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "start_date" bigint NOT NULL,
    "end_date" bigint NOT NULL,
    "tax_year" int,
    "budget_amount" bigint,
    "adjustment_deadline" bigint,
    "is_current" boolean NOT NULL DEFAULT FALSE,
    "is_calendar_year" boolean NOT NULL DEFAULT FALSE,
    "allow_adjusting_entries" boolean NOT NULL DEFAULT FALSE,
    "closed_at" bigint,
    "locked_at" bigint,
    "closed_by_id" varchar(100),
    "locked_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_fiscal_years" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fiscal_years_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_years_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_years_closed_by" FOREIGN KEY ("closed_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_fiscal_years_locked_by" FOREIGN KEY ("locked_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_fiscal_years_year" UNIQUE ("organization_id", "year")
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_fiscal_years_unique_current ON "fiscal_years"("organization_id")
WHERE
    "is_current" = TRUE;

CREATE INDEX IF NOT EXISTS idx_fiscal_years_bu_org ON "fiscal_years"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS idx_fiscal_years_created_updated ON "fiscal_years"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS idx_fiscal_years_status ON "fiscal_years"("status");

CREATE INDEX IF NOT EXISTS idx_fiscal_years_year ON "fiscal_years"("year");

CREATE INDEX IF NOT EXISTS idx_fiscal_years_current ON "fiscal_years"("is_current")
WHERE
    "is_current" = TRUE;

CREATE INDEX IF NOT EXISTS idx_fiscal_years_date_range ON "fiscal_years"("start_date", "end_date");

COMMENT ON TABLE "fiscal_years" IS 'Stores information about fiscal years';

-- 1. Search Vector Column
ALTER TABLE "fiscal_years"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_years_search_vector ON "fiscal_years" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION fiscal_years_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- UPDATED: Include year in search
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.year::text, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS fiscal_years_search_update ON "fiscal_years";

CREATE TRIGGER fiscal_years_search_update
    BEFORE INSERT OR UPDATE ON "fiscal_years"
    FOR EACH ROW
    EXECUTE FUNCTION fiscal_years_search_trigger();

--bun:split
UPDATE
    "fiscal_years"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(name, '')), 'A') || setweight(to_tsvector('english', COALESCE(year::text, '')), 'A') || setweight(to_tsvector('english', COALESCE(description, '')), 'B');

--bun:split
CREATE OR REPLACE FUNCTION enforce_single_current_fiscal_year()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF NEW.is_current = TRUE THEN
        UPDATE
            fiscal_years
        SET
            is_current = FALSE
        WHERE
            organization_id = NEW.organization_id
            AND id != NEW.id
            AND is_current = TRUE;
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE TRIGGER trigger_enforce_single_current_fiscal_year
    BEFORE INSERT OR UPDATE ON fiscal_years
    FOR EACH ROW
    EXECUTE FUNCTION enforce_single_current_fiscal_year();

--bun:split
ALTER TABLE "fiscal_years"
    ALTER COLUMN "status" SET STATISTICS 1000;

--bun:split
ALTER TABLE "fiscal_years"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "fiscal_years"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

