CREATE TYPE account_category_enum AS ENUM(
    'Asset',
    'Liability',
    'Equity',
    'Revenue',
    'CostOfRevenue',
    'Expense'
);

CREATE TABLE IF NOT EXISTS "account_types"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "category" account_category_enum NOT NULL,
    "color" varchar(10),
    "is_system" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_account_types" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_account_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_account_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_account_types_unique_code" ON "account_types"("organization_id", lower("code"));

CREATE INDEX IF NOT EXISTS "idx_account_types_bu_org" ON "account_types"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_account_types_created_updated" ON "account_types"("created_at", "updated_at");

COMMENT ON TABLE "account_types" IS 'Stores information about account types';

--bun:split
ALTER TABLE "account_types"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("code", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE("description", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_account_types_search_vector ON "account_types" USING GIN(search_vector);
