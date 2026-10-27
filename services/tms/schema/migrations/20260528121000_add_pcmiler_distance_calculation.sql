ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'PCMiler';

ALTER TABLE "shipment_moves"
    ADD COLUMN IF NOT EXISTS "distance_source" varchar(50),
    ADD COLUMN IF NOT EXISTS "distance_provider" varchar(50),
    ADD COLUMN IF NOT EXISTS "distance_calculated_at" bigint,
    ADD COLUMN IF NOT EXISTS "distance_route_signature" text,
    ADD COLUMN IF NOT EXISTS "distance_data_version" varchar(50),
    ADD COLUMN IF NOT EXISTS "distance_routing_type" varchar(50),
    ADD COLUMN IF NOT EXISTS "distance_units" varchar(50),
    ADD COLUMN IF NOT EXISTS "distance_metadata" jsonb;

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_distance_route_signature"
    ON "shipment_moves"("organization_id", "business_unit_id", "distance_route_signature");

CREATE TABLE IF NOT EXISTS "distance_profiles"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "is_default" boolean NOT NULL DEFAULT false,
    "provider" varchar(50) NOT NULL,
    "data_version" varchar(50) NOT NULL DEFAULT 'Current',
    "region" varchar(20) NOT NULL DEFAULT 'NA',
    "routing_type" varchar(50) NOT NULL DEFAULT 'Practical',
    "distance_units" varchar(50) NOT NULL DEFAULT 'Miles',
    "location_granularity" varchar(50) NOT NULL DEFAULT 'PostalCode',
    "profile_name" varchar(100),
    "highway_only" boolean NOT NULL DEFAULT false,
    "toll_roads" boolean NOT NULL DEFAULT true,
    "borders_open" boolean NOT NULL DEFAULT true,
    "include_toll_data" boolean NOT NULL DEFAULT false,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_distance_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_distance_profiles_status" CHECK ("status" IN ('Active', 'Inactive')),
    CONSTRAINT "ck_distance_profiles_provider" CHECK ("provider" IN ('PCMiler')),
    CONSTRAINT "ck_distance_profiles_default_active" CHECK ("is_default" = false OR "status" = 'Active')
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_profiles_default"
    ON "distance_profiles"("organization_id", "business_unit_id")
    WHERE "is_default" = true;

CREATE INDEX IF NOT EXISTS "idx_distance_profiles_tenant_status"
    ON "distance_profiles"("organization_id", "business_unit_id", "status", "updated_at" DESC);

INSERT INTO "distance_profiles"(
    "id",
    "organization_id",
    "business_unit_id",
    "name",
    "description",
    "status",
    "is_default",
    "provider",
    "data_version",
    "region",
    "routing_type",
    "distance_units",
    "location_granularity",
    "highway_only",
    "toll_roads",
    "borders_open",
    "include_toll_data"
)
SELECT
    CONCAT('dp_', replace(gen_random_uuid()::text, '-', '')),
    org.id,
    org.business_unit_id,
    'Default PC*Miler',
    'Default PC*Miler routing profile',
    'Active',
    true,
    'PCMiler',
    'Current',
    'NA',
    'Practical',
    'Miles',
    'PostalCode',
    false,
    true,
    true,
    false
FROM "organizations" org
WHERE NOT EXISTS (
    SELECT 1
    FROM "distance_profiles" dp
    WHERE dp.organization_id = org.id
      AND dp.business_unit_id = org.business_unit_id
      AND dp.is_default = true
);

UPDATE "integrations"
SET "configuration" = jsonb_strip_nulls(jsonb_build_object(
    'apiKey', "configuration" -> 'apiKey',
    'baseUrl', "configuration" -> 'baseUrl'
))
WHERE "type" = 'PCMiler';

CREATE TABLE IF NOT EXISTS "distance_calculation_runs"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "provider" varchar(50),
    "source" varchar(50) NOT NULL,
    "request_summary" jsonb,
    "response_summary" jsonb,
    "status" varchar(50) NOT NULL,
    "error_code" varchar(100),
    "error_message" text,
    "latency_millis" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_distance_calculation_runs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_calculation_runs_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_calculation_runs_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_distance_calculation_runs_shipment"
    ON "distance_calculation_runs"("organization_id", "business_unit_id", "shipment_id", "created_at" DESC);

CREATE TABLE IF NOT EXISTS "provider_location_resolutions"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "location_id" varchar(100) NOT NULL,
    "provider" varchar(50) NOT NULL,
    "normalized_address" jsonb,
    "place_id" text,
    "metadata" jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_provider_location_resolutions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_provider_location_resolutions_location" FOREIGN KEY ("location_id", "organization_id", "business_unit_id") REFERENCES "locations"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_provider_location_resolutions_location_provider"
    ON "provider_location_resolutions"("organization_id", "business_unit_id", "location_id", "provider");
