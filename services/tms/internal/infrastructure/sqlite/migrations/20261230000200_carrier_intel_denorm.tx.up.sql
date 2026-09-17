-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261230000200_carrier_intel_denorm.tx.up.sql

ALTER TABLE "carriers" ADD COLUMN "intel_risk_level" TEXT;

--bun:split

ALTER TABLE "carriers" ADD COLUMN "intel_review_required" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "carriers" ADD COLUMN "intel_blocking_count" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_carriers_intel_review_required" ON "carriers" ("organization_id", "business_unit_id")WHERE
    "intel_review_required";

--bun:split

ALTER TABLE "customers" ADD COLUMN "dot_number" TEXT;

--bun:split

ALTER TABLE "customers" ADD COLUMN "mc_number" TEXT;

--bun:split

ALTER TABLE "customers" ADD COLUMN "broker_vetting_enabled" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customers_broker_vetting" ON "customers" ("organization_id", "business_unit_id")WHERE
    "broker_vetting_enabled";
