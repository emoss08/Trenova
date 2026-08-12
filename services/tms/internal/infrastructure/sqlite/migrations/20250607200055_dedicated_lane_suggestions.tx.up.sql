-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250607200055_dedicated_lane_suggestions.tx.up.sql

CREATE TABLE IF NOT EXISTS "dedicated_lane_suggestions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "customer_id" TEXT NOT NULL,
    "origin_location_id" TEXT NOT NULL,
    "destination_location_id" TEXT NOT NULL,
    "service_type_id" TEXT,
    "shipment_type_id" TEXT,
    "trailer_type_id" TEXT,
    "tractor_type_id" TEXT,
    "confidence_score" REAL NOT NULL,
    "frequency_count" INTEGER NOT NULL,
    "average_freight_charge" REAL,
    "total_freight_value" REAL,
    "last_shipment_date" INTEGER NOT NULL,
    "first_shipment_date" INTEGER NOT NULL,
    "suggested_name" TEXT NOT NULL,
    "pattern_details" TEXT NOT NULL DEFAULT '{}',
    "created_dedicated_lane_id" TEXT,
    "processed_by_id" TEXT,
    "processed_at" INTEGER,
    "expires_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_dedicated_lane_suggestions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_dedicated_lane_suggestions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lane_suggestions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lane_suggestions_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_dedicated_lane_suggestions_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lane_suggestions_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_dedicated_lane_suggestions_service_type" FOREIGN KEY ("service_type_id", "business_unit_id", "organization_id") REFERENCES "service_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_shipment_type" FOREIGN KEY ("shipment_type_id", "business_unit_id", "organization_id") REFERENCES "shipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_trailer_type" FOREIGN KEY ("trailer_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_tractor_type" FOREIGN KEY ("tractor_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_created_lane" FOREIGN KEY ("created_dedicated_lane_id", "organization_id", "business_unit_id") REFERENCES "dedicated_lanes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_dedicated_lane_suggestions_processed_by" FOREIGN KEY ("processed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_dedicated_lane_suggestions_different_locations" CHECK (origin_location_id != destination_location_id),
    CONSTRAINT "chk_dedicated_lane_suggestions_confidence_score" CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    CONSTRAINT "chk_dedicated_lane_suggestions_frequency_count" CHECK (frequency_count >= 1),
    CONSTRAINT "chk_dedicated_lane_suggestions_processed_logic" CHECK ((status = 'Pending' AND processed_by_id IS NULL AND processed_at IS NULL) OR (status IN ('Accepted', 'Rejected') AND processed_by_id IS NOT NULL AND processed_at IS NOT NULL) OR (status = 'Expired')),
    CONSTRAINT "chk_dedicated_lane_suggestions_acceptance_logic" CHECK ((status = 'Accepted' AND created_dedicated_lane_id IS NOT NULL) OR (status != 'Accepted' AND created_dedicated_lane_id IS NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_business_unit_org" ON "dedicated_lane_suggestions" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_pending" ON "dedicated_lane_suggestions" ("organization_id", "business_unit_id", "confidence_score" DESC, "created_at" DESC)WHERE
    status = 'Pending';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_customer" ON "dedicated_lane_suggestions" ("customer_id", "business_unit_id", "organization_id", "status", "confidence_score" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_locations" ON "dedicated_lane_suggestions" ("origin_location_id", "destination_location_id", "customer_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_expiration" ON "dedicated_lane_suggestions" ("expires_at", "status")WHERE
    status = 'Pending';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_analytics" ON "dedicated_lane_suggestions" ("processed_at", "status", "confidence_score")WHERE
    processed_at IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_confidence" ON "dedicated_lane_suggestions" ("confidence_score" DESC, "frequency_count" DESC)WHERE
    status = 'Pending' AND confidence_score >= 0.7;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_equipment" ON "dedicated_lane_suggestions" ("tractor_type_id", "trailer_type_id", "service_type_id", "shipment_type_id")WHERE
    tractor_type_id IS NOT NULL OR trailer_type_id IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_timestamps" ON "dedicated_lane_suggestions" ("created_at" DESC, "updated_at" DESC);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_unique_pending_pattern" ON "dedicated_lane_suggestions" ("customer_id", "origin_location_id", "destination_location_id", "organization_id", COALESCE("service_type_id", ''), COALESCE("shipment_type_id", ''), COALESCE("trailer_type_id", ''), COALESCE("tractor_type_id", ''))WHERE
    status = 'Pending';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_dedicated_lane_suggestions_high_value" ON "dedicated_lane_suggestions" ("total_freight_value" DESC, "frequency_count" DESC, "confidence_score" DESC)WHERE
    status = 'Pending' AND total_freight_value IS NOT NULL AND total_freight_value > 10000;
