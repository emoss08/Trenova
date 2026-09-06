-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261011000001_pto_followons.tx.up.sql

ALTER TABLE "pto_policy_rules" ADD COLUMN "tiers" TEXT NOT NULL DEFAULT '[]';

--bun:split

ALTER TABLE "pto_policy_rules" ADD COLUMN "on_termination" TEXT NOT NULL DEFAULT 'Forfeit';

--bun:split

ALTER TABLE "workers" ADD COLUMN "leave_type" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS "org_holidays"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "holiday_date" INTEGER NOT NULL,
    "kind" TEXT NOT NULL DEFAULT 'Holiday',
    "recurs_annually" INTEGER NOT NULL DEFAULT 0,
    "description" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_org_holidays" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_org_holidays_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_org_holidays_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_org_holidays_date" CHECK ("holiday_date" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_org_holidays_date_kind" ON "org_holidays" ("organization_id", "business_unit_id", "holiday_date", "kind");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_org_holidays_date" ON "org_holidays" ("organization_id", "business_unit_id", "holiday_date");
