-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250814055823_add_shipment_holds.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_holds"(
    "id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "severity" TEXT NOT NULL DEFAULT 'Advisory',
    "reason_code" TEXT,
    "notes" TEXT,
    "source" TEXT NOT NULL DEFAULT 'User',
    "blocks_dispatch" INTEGER NOT NULL DEFAULT 0,
    "blocks_delivery" INTEGER NOT NULL DEFAULT 0,
    "blocks_billing" INTEGER NOT NULL DEFAULT 0,
    "visible_to_customer" INTEGER NOT NULL DEFAULT 0,
    "metadata" TEXT,
    "started_at" INTEGER NOT NULL CHECK ("started_at" > 0),
    "released_at" INTEGER,
    "created_by_id" TEXT,
    "released_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "version" INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT "pk_shipment_holds" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipment_holds_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_holds_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_holds_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_holds_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_shipment_holds_released_by" FOREIGN KEY ("released_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_shipment_holds_released_ge_started" CHECK ("released_at" IS NULL OR "released_at" >= "started_at"),
    CONSTRAINT "ck_shipment_holds_blocking_requires_flags" CHECK ("severity" <> 'Blocking' OR ("blocks_dispatch" OR "blocks_delivery" OR "blocks_billing"))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_bu_org" ON "shipment_holds" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_type" ON "shipment_holds" ("type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_source" ON "shipment_holds" ("source");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_holds_active_by_shipment" ON "shipment_holds" ("shipment_id")WHERE
    "released_at" IS NULL;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "ux_shipment_holds_active_by_type" ON "shipment_holds" ("shipment_id", "organization_id", "business_unit_id", "type")WHERE
    "released_at" IS NULL;
