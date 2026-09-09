--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "shipment_move_jurisdiction_miles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "country_code" varchar(2) NOT NULL,
    "jurisdiction_code" varchar(5) NOT NULL,
    "sequence" integer NOT NULL DEFAULT 0,
    "distance" double precision NOT NULL DEFAULT 0,
    "distance_units" varchar(50) NOT NULL DEFAULT 'Miles',
    "toll_distance" double precision,
    "ferry_distance" double precision,
    "loaded" boolean NOT NULL DEFAULT TRUE,
    "source" varchar(50) NOT NULL DEFAULT 'RouteCalculation',
    "provider" varchar(50),
    "data_version" varchar(50),
    "distance_profile_id" varchar(100),
    "calculated_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_shipment_move_jurisdiction_miles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_shipment_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_move_jurisdiction_miles_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_distance" CHECK ("distance" >= 0),
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_units" CHECK ("distance_units" IN ('Miles', 'Kilometers')),
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_source" CHECK ("source" IN ('RouteCalculation', 'Manual')),
    CONSTRAINT "chk_shipment_move_jurisdiction_miles_country" CHECK ("country_code" = upper("country_code") AND length("country_code") = 2)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_shipment_move_jurisdiction_miles_move_jurisdiction" ON "shipment_move_jurisdiction_miles"("organization_id", "business_unit_id", "shipment_move_id", "country_code", "jurisdiction_code");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_move_jurisdiction_miles_jurisdiction" ON "shipment_move_jurisdiction_miles"("organization_id", "business_unit_id", "country_code", "jurisdiction_code", "calculated_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_move_jurisdiction_miles_shipment" ON "shipment_move_jurisdiction_miles"("organization_id", "business_unit_id", "shipment_id");

--bun:split
COMMENT ON TABLE shipment_move_jurisdiction_miles IS 'One row per move and jurisdiction from the routing provider''s state-by-state report (or a manual entry). Keyed by country and jurisdiction code rather than an IFTA jurisdiction row because the provider returns codes and the shipment domain does not depend on the IFTA reference table; an unknown code is stored, not rejected.';

--bun:split
ALTER TABLE "stored_mileages"
    ADD COLUMN IF NOT EXISTS "jurisdiction_distances" jsonb;

--bun:split
COMMENT ON COLUMN stored_mileages.jurisdiction_distances IS 'Per-jurisdiction breakdown [{country, code, distance, toll, ferry}] captured with the cached route when the state report was requested; NULL when the cache entry was written without one.';

--bun:split
ALTER TABLE "distance_controls"
    ADD COLUMN IF NOT EXISTS "capture_jurisdiction_miles" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN distance_controls.capture_jurisdiction_miles IS 'Ask the routing provider for the state-by-state mileage report on every move route and persist per-jurisdiction rows for IFTA. Off by default because it may be billed as an additional transaction per route.';
