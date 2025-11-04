CREATE TYPE period_type_enum AS ENUM(
    'Month',
    'Quarter',
    'Year'
);

CREATE TYPE period_status_enum AS ENUM(
    'Open',
    'Closed',
    'Locked'
);

CREATE TABLE IF NOT EXISTS "fiscal_periods"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "fiscal_year_id" varchar(100) NOT NULL,
    "period_number" int NOT NULL,
    "period_type" period_type_enum NOT NULL DEFAULT 'Month',
    "name" varchar(100) NOT NULL,
    "start_date" bigint NOT NULL,
    "end_date" bigint NOT NULL,
    "status" period_status_enum NOT NULL DEFAULT 'Open',
    "closed_at" bigint,
    "closed_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_fiscal_periods" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fiscal_periods_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_fiscal_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_closed_by" FOREIGN KEY ("closed_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_fiscal_periods_year_number" UNIQUE ("fiscal_year_id", "period_number"),
    CONSTRAINT "chk_fiscal_periods_dates" CHECK ("end_date" > "start_date")
);

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_fiscal_year ON "fiscal_periods"("fiscal_year_id");

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_bu_org ON "fiscal_periods"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_dates ON "fiscal_periods"("start_date", "end_date");

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_status ON "fiscal_periods"("status");

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_created_updated ON "fiscal_periods"("created_at", "updated_at");

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_period_number ON "fiscal_periods"("period_number");

COMMENT ON TABLE "fiscal_periods" IS 'Stores information about fiscal periods within fiscal years';

-- 1. Search Vector Column
ALTER TABLE "fiscal_periods"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_search_vector ON "fiscal_periods" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION fiscal_periods_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.period_number::text, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.status::text, '')), 'C');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS fiscal_periods_search_update ON "fiscal_periods";

CREATE TRIGGER fiscal_periods_search_update
    BEFORE INSERT OR UPDATE ON "fiscal_periods"
    FOR EACH ROW
    EXECUTE FUNCTION fiscal_periods_search_trigger();

--bun:split
UPDATE
    "fiscal_periods"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(name, '')), 'A') || setweight(to_tsvector('english', COALESCE(period_number::text, '')), 'A') || setweight(to_tsvector('english', COALESCE(status::text, '')), 'C');

--bun:split
ALTER TABLE "fiscal_periods"
    ALTER COLUMN "status" SET STATISTICS 1000;

--bun:split
ALTER TABLE "fiscal_periods"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "fiscal_periods"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "fiscal_periods"
    ALTER COLUMN "fiscal_year_id" SET STATISTICS 1000;

