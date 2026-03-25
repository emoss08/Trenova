CREATE TYPE period_type_enum AS ENUM(
    'Month',
    'Quarter',
    'Week',
    'Adjusting'
);

CREATE TYPE period_status_enum AS ENUM(
    'Inactive',
    'Open',
    'Locked',
    'Closed',
    'PermanentlyClosed'
);

CREATE TABLE IF NOT EXISTS "fiscal_periods"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "fiscal_year_id" varchar(100) NOT NULL,
    "period_number" int NOT NULL,
    "period_type" period_type_enum NOT NULL DEFAULT 'Month',
    "status" period_status_enum NOT NULL DEFAULT 'Inactive',
    "name" varchar(100) NOT NULL,
    "start_date" bigint NOT NULL,
    "end_date" bigint NOT NULL,
    "is_adjusting" boolean NOT NULL DEFAULT FALSE,
    "allow_adjusting_entries" boolean NOT NULL DEFAULT FALSE,
    "adjustment_deadline" bigint,
    "locked_at" bigint,
    "locked_by_id" varchar(100),
    "closed_at" bigint,
    "closed_by_id" varchar(100),
    "reopened_at" bigint,
    "reopened_by_id" varchar(100),
    "reopen_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_fiscal_periods" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fiscal_periods_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_fiscal_year" FOREIGN KEY ("fiscal_year_id", "organization_id", "business_unit_id") REFERENCES "fiscal_years"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_fiscal_periods_closed_by" FOREIGN KEY ("closed_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_fiscal_periods_locked_by" FOREIGN KEY ("locked_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_fiscal_periods_reopened_by" FOREIGN KEY ("reopened_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_fiscal_periods_year_number" UNIQUE ("fiscal_year_id", "period_number"),
    CONSTRAINT "chk_fiscal_periods_dates" CHECK ("end_date" > "start_date"),
    CONSTRAINT "no_overlapping_periods"
    EXCLUDE USING gist("organization_id" WITH =, int8range("start_date", "end_date", '[]'
) WITH &&)
);

--bun:split
ALTER TABLE "fiscal_periods"
    ADD COLUMN "search_vector" tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('english', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('english', COALESCE("period_number"::text, '')), 'A')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_fiscal_year ON "fiscal_periods"("fiscal_year_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_bu_org ON "fiscal_periods"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_dates ON "fiscal_periods"("start_date", "end_date");

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_status ON "fiscal_periods"("status");

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_updated_at ON "fiscal_periods"("updated_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_period_number ON "fiscal_periods"("period_number");

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_is_adjusting ON "fiscal_periods"("is_adjusting")
WHERE
    "is_adjusting" = TRUE;

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_open ON "fiscal_periods"("organization_id", "business_unit_id", "start_date", "end_date")
WHERE
    "status" = 'Open';

--bun:split
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_search_vector ON "fiscal_periods" USING GIN("search_vector");

--bun:split
CREATE STATISTICS IF NOT EXISTS fiscal_periods_org_status_stats(dependencies) ON "organization_id", "status" FROM "fiscal_periods";

--bun:split
CREATE STATISTICS IF NOT EXISTS fiscal_periods_org_bu_stats(dependencies) ON "organization_id", "business_unit_id" FROM "fiscal_periods";

--bun:split
COMMENT ON TABLE "fiscal_periods" IS 'Stores information about fiscal periods within fiscal years';

