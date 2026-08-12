-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250815033255_add_hold_reason.tx.up.sql

CREATE TABLE IF NOT EXISTS "hold_reasons"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "label" TEXT NOT NULL,
    "description" TEXT,
    "default_severity" TEXT NOT NULL DEFAULT 'Advisory',
    "default_blocks_dispatch" INTEGER NOT NULL DEFAULT 0,
    "default_blocks_delivery" INTEGER NOT NULL DEFAULT 0,
    "default_blocks_billing" INTEGER NOT NULL DEFAULT 0,
    "default_visible_to_customer" INTEGER NOT NULL DEFAULT 0,
    "active" INTEGER NOT NULL DEFAULT 1,
    "sort_order" INTEGER NOT NULL DEFAULT 100,
    "external_map" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_hold_reasons" PRIMARY KEY ("id", "organization_id"),
    CONSTRAINT "fk_hr_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hr_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ux_hr_org_bu_type_code" UNIQUE ("organization_id", "business_unit_id", "type", "code")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hr_org_bu_type_active" ON "hold_reasons" ("organization_id", "business_unit_id", "type")WHERE
    active = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hr_code_type_label" ON "hold_reasons" ("code", "type", "label");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_hr_status" ON "hold_reasons" ("active");
