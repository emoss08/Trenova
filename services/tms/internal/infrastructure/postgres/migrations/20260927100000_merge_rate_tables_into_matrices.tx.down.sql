-- Restores the rate table schema so the code at this revision can run again.
--
-- The data stays where the up migration put it: the migrated matrices are not
-- unwound, because nothing distinguishes a migrated single-axis matrix from
-- one somebody built by hand after the merge, and guessing would move rows a
-- person created. The rates are all still present and readable as matrices.
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
