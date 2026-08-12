-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251029201602_add_account_types.tx.up.sql

CREATE TABLE IF NOT EXISTS "account_types"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "category" TEXT NOT NULL,
    "color" TEXT,
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_account_types" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_account_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_account_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_account_types_unique_code" ON "account_types" ("organization_id", lower("code"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_account_types_bu_org" ON "account_types" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_account_types_created_updated" ON "account_types" ("created_at", "updated_at");
