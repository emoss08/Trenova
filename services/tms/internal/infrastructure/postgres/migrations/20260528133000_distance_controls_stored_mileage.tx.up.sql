CREATE TABLE IF NOT EXISTS "distance_controls"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "store_mileage" boolean NOT NULL DEFAULT true,
    "stored_distance_units" varchar(50) NOT NULL DEFAULT 'Miles',
    "postal_code_fallback_to_city" boolean NOT NULL DEFAULT true,
    "auto_create_stored_mileage" boolean NOT NULL DEFAULT true,
    "loaded_move_distance_profile_id" varchar(100) NOT NULL,
    "empty_move_distance_profile_id" varchar(100) NOT NULL,
    "pay_distance_profile_id" varchar(100) NOT NULL,
    "billing_distance_profile_id" varchar(100) NOT NULL,
    "fuel_distance_profile_id" varchar(100) NOT NULL,
    "eta_out_of_route_distance_profile_id" varchar(100) NOT NULL,
    "distance_calculator_shortest_distance_profile_id" varchar(100) NOT NULL,
    "distance_calculator_practical_distance_profile_id" varchar(100) NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_distance_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_distance_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_distance_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_distance_controls_stored_units" CHECK ("stored_distance_units" IN ('Miles', 'Kilometers'))
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_distance_controls_tenant"
    ON "distance_controls"("organization_id", "business_unit_id");

INSERT INTO "distance_controls"(
    "id",
    "organization_id",
    "business_unit_id",
    "loaded_move_distance_profile_id",
    "empty_move_distance_profile_id",
    "pay_distance_profile_id",
    "billing_distance_profile_id",
    "fuel_distance_profile_id",
    "eta_out_of_route_distance_profile_id",
    "distance_calculator_shortest_distance_profile_id",
    "distance_calculator_practical_distance_profile_id"
)
SELECT
    CONCAT('dc_', replace(gen_random_uuid()::text, '-', '')),
    dp.organization_id,
    dp.business_unit_id,
    dp.id,
    dp.id,
    dp.id,
    dp.id,
    dp.id,
    dp.id,
    COALESCE(shortest.id, dp.id),
    dp.id
FROM "distance_profiles" dp
LEFT JOIN LATERAL (
    SELECT sp.id
    FROM "distance_profiles" sp
    WHERE sp.organization_id = dp.organization_id
      AND sp.business_unit_id = dp.business_unit_id
      AND sp.status = 'Active'
      AND lower(sp.routing_type) = 'shortest'
    ORDER BY sp.updated_at DESC
    LIMIT 1
) shortest ON true
WHERE dp.is_default = true
  AND NOT EXISTS (
      SELECT 1
      FROM "distance_controls" dc
      WHERE dc.organization_id = dp.organization_id
        AND dc.business_unit_id = dp.business_unit_id
  );

CREATE TABLE IF NOT EXISTS "stored_mileages"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "origin_key" jsonb NOT NULL,
    "destination_key" jsonb NOT NULL,
    "intermediate_keys" jsonb,
    "route_signature" text NOT NULL,
    "route_hash" varchar(64) NOT NULL,
    "distance" double precision NOT NULL,
    "distance_units" varchar(50) NOT NULL,
    "provider" varchar(50) NOT NULL,
    "source" varchar(50) NOT NULL,
    "routing_type" varchar(50) NOT NULL,
    "method" varchar(50) NOT NULL,
    "location_granularity" varchar(50) NOT NULL,
    "data_version" varchar(50) NOT NULL,
    "distance_profile_id" varchar(100) NOT NULL,
    "distance_profile_name" varchar(100),
    "hazmat" boolean NOT NULL DEFAULT false,
    "hazmat_types" text[],
    "hazmat_signature" text NOT NULL DEFAULT 'none',
    "provider_metadata" jsonb,
    "hit_count" bigint NOT NULL DEFAULT 0,
    "last_used_at" bigint,
    "last_calculated_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_stored_mileages" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_stored_mileages_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_stored_mileages_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_stored_mileages_status" CHECK ("status" IN ('Active', 'Inactive')),
    CONSTRAINT "ck_stored_mileages_distance" CHECK ("distance" >= 0),
    CONSTRAINT "ck_stored_mileages_units" CHECK ("distance_units" IN ('Miles', 'Kilometers'))
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_stored_mileages_active_policy"
    ON "stored_mileages"(
        "organization_id",
        "business_unit_id",
        "route_hash",
        "distance_units",
        "routing_type",
        "method",
        "distance_profile_id",
        "hazmat_signature"
    )
    WHERE "status" = 'Active';

CREATE INDEX IF NOT EXISTS "idx_stored_mileages_tenant_status"
    ON "stored_mileages"("organization_id", "business_unit_id", "status", "updated_at" DESC);
