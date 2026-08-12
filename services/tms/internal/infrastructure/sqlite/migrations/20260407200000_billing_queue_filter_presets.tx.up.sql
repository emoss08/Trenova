-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260407200000_billing_queue_filter_presets.tx.up.sql

CREATE TABLE IF NOT EXISTS billing_queue_filter_presets(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "filters" TEXT NOT NULL DEFAULT '{}',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_billing_queue_filter_presets" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_billing_queue_filter_presets_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_filter_presets_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_filter_presets_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_billing_queue_filter_presets_user ON billing_queue_filter_presets ("user_id", "organization_id", "business_unit_id");
