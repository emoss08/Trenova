ALTER TABLE "distance_overrides"
    ADD COLUMN IF NOT EXISTS "route_signature" text;

UPDATE "distance_overrides"
SET
    "route_signature" = concat_ws('|', COALESCE("customer_id", '*'), concat_ws('>', "origin_location_id", "destination_location_id"))
WHERE
    "route_signature" IS NULL;

ALTER TABLE "distance_overrides"
    ALTER COLUMN "route_signature" SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_overrides_route_signature" ON "distance_overrides"(
    "organization_id",
    "business_unit_id",
    "route_signature"
);

CREATE TABLE IF NOT EXISTS "distance_override_stops"(
    "distance_override_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "stop_order" integer NOT NULL,
    "location_id" varchar(100) NOT NULL,
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_override_stops_location" ON "distance_override_stops"(
    "distance_override_id",
    "organization_id",
    "business_unit_id",
    "location_id"
);

CREATE INDEX IF NOT EXISTS "idx_distance_override_stops_lookup" ON "distance_override_stops"(
    "distance_override_id",
    "organization_id",
    "business_unit_id",
    "stop_order"
);
