-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250628004846_add_consolidations.tx.up.sql

CREATE TABLE IF NOT EXISTS "consolidation_groups"(
    "id" TEXT NOT NULL,
    "consolidation_number" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'New',
    "canceled_by_id" TEXT,
    "canceled_at" INTEGER,
    "cancel_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_consolidation_groups" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_consolidation_groups_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_consolidation_groups_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_consolidation_groups_canceled_by" FOREIGN KEY ("canceled_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_consolidation_groups_consolidation_number" ON "consolidation_groups" ("consolidation_number", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_status" ON "consolidation_groups" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_business_unit" ON "consolidation_groups" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_bu_org_status_created_at" ON "consolidation_groups" ("business_unit_id", "organization_id", "status", "created_at" DESC);

--bun:split

ALTER TABLE "customers" ADD COLUMN "allow_consolidation" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "customers" ADD COLUMN "exclusive_consolidation" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "customers" ADD COLUMN "consolidation_priority" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "consolidation_group_id" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS "consolidation_settings"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "max_pickup_distance" TEXT NOT NULL DEFAULT 25,
    "max_delivery_distance" TEXT NOT NULL DEFAULT 25,
    "max_route_detour" TEXT NOT NULL DEFAULT 15,
    "max_time_window_gap" INTEGER NOT NULL DEFAULT 240,
    "min_time_buffer" INTEGER NOT NULL DEFAULT 30,
    "max_shipments_per_group" INTEGER NOT NULL DEFAULT 3,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_consolidation_settings" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_consolidation_settings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_consolidation_settings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_consolidation_groups_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_consolidation_settings_business_unit" ON "consolidation_settings" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_consolidation_settings_created_at" ON "consolidation_settings" ("created_at", "updated_at");
