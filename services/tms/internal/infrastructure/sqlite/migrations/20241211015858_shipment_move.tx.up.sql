-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211015858_shipment_move.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_moves" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'New',
    "loaded" INTEGER NOT NULL DEFAULT 1,
    "sequence" INTEGER NOT NULL DEFAULT 0 CHECK ("sequence" >= 0),
    "distance" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_moves" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipment_moves_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_moves_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_moves_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_created_at" ON "shipment_moves" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_status" ON "shipment_moves" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_business_unit" ON "shipment_moves" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_shipment" ON "shipment_moves" ("shipment_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_sequence" ON "shipment_moves" ("sequence");
