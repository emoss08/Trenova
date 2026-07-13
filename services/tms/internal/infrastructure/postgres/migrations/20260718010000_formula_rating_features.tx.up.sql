ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS "breakdown_definitions" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "max_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "submitted_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "submitted_at" bigint,
    ADD COLUMN IF NOT EXISTS "approved_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "approved_at" bigint,
    ADD COLUMN IF NOT EXISTS "review_comment" text;

--bun:split
ALTER TABLE "formula_template_versions"
    ADD COLUMN IF NOT EXISTS "breakdown_definitions" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "max_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "effective_from" bigint;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_template_versions_effective"
    ON "formula_template_versions"("template_id", "organization_id", "business_unit_id", "effective_from" DESC)
    WHERE "effective_from" IS NOT NULL;

--bun:split
CREATE TYPE "rate_table_lookup_type_enum" AS ENUM('Exact', 'Range');

--bun:split
CREATE TABLE IF NOT EXISTS "rate_tables"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "key" varchar(64) NOT NULL,
    "description" text,
    "lookup_type" rate_table_lookup_type_enum NOT NULL,
    "active" boolean NOT NULL DEFAULT TRUE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce("name", '')), 'A') ||
        setweight(to_tsvector('english', coalesce("key", '')), 'A') ||
        setweight(to_tsvector('english', coalesce("description", '')), 'B')
    ) STORED,
    CONSTRAINT "pk_rate_tables" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_tables_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_tables_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_tables_key" ON "rate_tables"("organization_id", "business_unit_id", "key");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_tables_bu_org" ON "rate_tables"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_tables_search" ON "rate_tables" USING gin("search_vector");

--bun:split
CREATE TABLE IF NOT EXISTS "rate_table_entries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_table_id" varchar(100) NOT NULL,
    "match_key" varchar(100),
    "range_min" numeric(19, 4),
    "range_max" numeric(19, 4),
    "value" numeric(19, 4) NOT NULL,
    "sort_order" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_table_entries" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_rate_table_entries_rate_table" FOREIGN KEY ("rate_table_id", "business_unit_id", "organization_id") REFERENCES "rate_tables"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_table_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_table_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_table_entries_table" ON "rate_table_entries"("rate_table_id", "organization_id", "business_unit_id");
