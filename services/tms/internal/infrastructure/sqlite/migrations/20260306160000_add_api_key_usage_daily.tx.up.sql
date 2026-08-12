-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260306160000_add_api_key_usage_daily.tx.up.sql

CREATE TABLE IF NOT EXISTS "api_key_usage_daily" (
    "api_key_id" TEXT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "usage_date" TEXT NOT NULL,
    "request_count" INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY ("api_key_id", "usage_date")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_api_key_usage_daily_org_bu_date"
    ON "api_key_usage_daily" ("organization_id", "business_unit_id", "usage_date");
