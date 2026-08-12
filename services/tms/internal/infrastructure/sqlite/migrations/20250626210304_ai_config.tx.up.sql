-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250626210304_ai_config.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_configs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "api_key" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_configs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_ai_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_ai_configs_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_configs_business_unit" ON "ai_configs" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_configs_created_at" ON "ai_configs" ("created_at", "updated_at");
