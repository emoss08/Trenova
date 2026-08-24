-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260905000000_cargo_dimensions_and_decks.tx.up.sql

ALTER TABLE "equipment_types" ADD COLUMN "deck_type" TEXT;

--bun:split

ALTER TABLE "equipment_types" ADD COLUMN "deck_height_inches" REAL;

--bun:split

ALTER TABLE "equipment_types" ADD COLUMN "well_length_feet" REAL;

--bun:split

ALTER TABLE "equipment_types" ADD COLUMN "axle_count" INTEGER;

--bun:split

ALTER TABLE "commodities" ADD COLUMN "length_per_unit" REAL;

--bun:split

ALTER TABLE "commodities" ADD COLUMN "width_per_unit" REAL;

--bun:split

ALTER TABLE "commodities" ADD COLUMN "height_per_unit" REAL;

--bun:split

ALTER TABLE "shipment_commodities" ADD COLUMN "length_feet" REAL;

--bun:split

ALTER TABLE "shipment_commodities" ADD COLUMN "width_feet" REAL;

--bun:split

ALTER TABLE "shipment_commodities" ADD COLUMN "height_feet" REAL;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "envelope_length_feet" REAL;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "envelope_width_feet" REAL;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "envelope_height_feet" REAL;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "envelope_overall_height_feet" REAL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_envelope_oversize" ON "shipments" ("organization_id", "business_unit_id", "envelope_width_feet" DESC, "envelope_overall_height_feet" DESC)WHERE
    "envelope_width_feet" IS NOT NULL;
