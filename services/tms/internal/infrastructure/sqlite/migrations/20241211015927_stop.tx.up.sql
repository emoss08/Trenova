-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211015927_stop.tx.up.sql

CREATE TABLE IF NOT EXISTS "stops"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_move_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'New',
    "type" TEXT NOT NULL DEFAULT 'Pickup',
    "sequence" INTEGER NOT NULL DEFAULT 0,
    "pieces" INTEGER,
    "weight" INTEGER,
    "planned_arrival" INTEGER NOT NULL,
    "planned_departure" INTEGER NOT NULL,
    "actual_arrival" INTEGER,
    "actual_departure" INTEGER,
    "address_line" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_stops" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_stops_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stops_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_stops_planned_times" CHECK (planned_departure >= planned_arrival),
    CONSTRAINT "chk_stops_actual_times" CHECK (actual_departure >= actual_arrival),
    CONSTRAINT "chk_stops_pieces_positive" CHECK (pieces IS NULL OR pieces >= 0),
    CONSTRAINT "chk_stops_weight_positive" CHECK (weight IS NULL OR weight >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_stops_active" ON "stops" ("organization_id", "business_unit_id", "status")WHERE
    status != 'Canceled';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_stops_common_queries" ON "stops" ("organization_id", "business_unit_id", "created_at", "status", "type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_stops_timestamps" ON "stops" ("created_at", "updated_at");
