-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260227100000_add_distance_override_stops.tx.up.sql

ALTER TABLE "distance_overrides" ADD COLUMN "route_signature" TEXT;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_overrides_route_signature" ON "distance_overrides" (
    "organization_id",
    "business_unit_id",
    "route_signature"
);

--bun:split

CREATE TABLE IF NOT EXISTS "distance_override_stops"(
    "distance_override_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "stop_order" INTEGER NOT NULL,
    "location_id" TEXT NOT NULL,
    CONSTRAINT "pk_distance_override_stops" PRIMARY KEY (
        "distance_override_id",
        "organization_id",
        "business_unit_id",
        "stop_order"
    ),
    CONSTRAINT "fk_distance_override_stops_distance_override" FOREIGN KEY (
        "distance_override_id",
        "organization_id",
        "business_unit_id"
    ) REFERENCES "distance_overrides"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_override_stops_location" FOREIGN KEY (
        "location_id",
        "business_unit_id",
        "organization_id"
    ) REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_distance_override_stops_positive_order" CHECK ("stop_order" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_override_stops_location" ON "distance_override_stops" (
    "distance_override_id",
    "organization_id",
    "business_unit_id",
    "location_id"
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_distance_override_stops_lookup" ON "distance_override_stops" (
    "distance_override_id",
    "organization_id",
    "business_unit_id",
    "stop_order"
);
