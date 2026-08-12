-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250227003617_shipment_control.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "max_shipment_weight_limit" INTEGER NOT NULL DEFAULT 80000,
    "auto_delay_shipments" INTEGER NOT NULL DEFAULT 0,
    "auto_delay_shipments_threshold" INTEGER,
    "track_detention_time" INTEGER NOT NULL DEFAULT 0,
    "auto_generate_detention_charges" INTEGER NOT NULL DEFAULT 0,
    "detention_threshold" INTEGER NOT NULL DEFAULT 30,
    "auto_cancel_shipments" INTEGER NOT NULL DEFAULT 0,
    "auto_cancel_shipments_threshold" INTEGER NOT NULL DEFAULT 30,
    "track_customer_rejections" INTEGER NOT NULL DEFAULT 0,
    "check_for_duplicate_bols" INTEGER NOT NULL DEFAULT 1,
    "allow_move_removals" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipment_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_shipment_controls_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_controls_business_unit" ON "shipment_controls" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_controls_created_at" ON "shipment_controls" ("created_at", "updated_at");

--bun:split

ALTER TABLE "shipment_controls" ADD COLUMN "check_hazmat_segregation" INTEGER NOT NULL DEFAULT 1;
