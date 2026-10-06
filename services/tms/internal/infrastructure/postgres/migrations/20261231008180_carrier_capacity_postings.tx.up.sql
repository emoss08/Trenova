-- A carrier says it has trucks: where they are, where they want to go, when,
-- with what equipment and at what rate. Postings feed the shipment board's
-- carrier capacity strip and the coverage suggestions for uncovered loads.
CREATE TABLE IF NOT EXISTS "carrier_capacity_postings"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "origin_location_id" varchar(100),
    "origin_state_id" varchar(100),
    "origin_radius_miles" integer,
    "destination_state_id" varchar(100),
    "equipment_type_id" varchar(100),
    "available_from" bigint NOT NULL,
    "available_to" bigint NOT NULL,
    "truck_count" integer NOT NULL DEFAULT 1,
    "rate_method" varchar(20) NOT NULL DEFAULT 'PerMile',
    "rate" numeric(19, 4),
    "source" varchar(20) NOT NULL DEFAULT 'Manual',
    "notes" text,
    "search_vector" tsvector GENERATED ALWAYS AS (to_tsvector('english', COALESCE("notes", ''))) STORED,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_carrier_capacity_postings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_carrier_capacity_postings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_capacity_postings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_capacity_postings_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_carrier_capacity_postings_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_capacity_postings_origin_state" FOREIGN KEY ("origin_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_capacity_postings_destination_state" FOREIGN KEY ("destination_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_carrier_capacity_postings_equipment_type" FOREIGN KEY ("equipment_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "ck_carrier_capacity_postings_rate_method" CHECK ("rate_method" IN ('Flat', 'PerMile')),
    CONSTRAINT "ck_carrier_capacity_postings_source" CHECK ("source" IN ('Manual', 'Email', 'EDI')),
    CONSTRAINT "ck_carrier_capacity_postings_origin" CHECK ("origin_location_id" IS NOT NULL OR "origin_state_id" IS NOT NULL),
    CONSTRAINT "ck_carrier_capacity_postings_radius" CHECK ("origin_radius_miles" IS NULL OR ("origin_radius_miles" BETWEEN 1 AND 1000 AND "origin_location_id" IS NOT NULL)),
    CONSTRAINT "ck_carrier_capacity_postings_window" CHECK ("available_to" > "available_from"),
    CONSTRAINT "ck_carrier_capacity_postings_trucks" CHECK ("truck_count" BETWEEN 1 AND 999),
    CONSTRAINT "ck_carrier_capacity_postings_rate" CHECK ("rate" IS NULL OR "rate" >= 0)
);

--bun:split
-- The board reads the postings still open, newest window last.
CREATE INDEX IF NOT EXISTS "idx_carrier_capacity_postings_open" ON "carrier_capacity_postings"("organization_id", "business_unit_id", "available_to", "available_from");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_capacity_postings_carrier" ON "carrier_capacity_postings"("organization_id", "business_unit_id", "carrier_id", "available_to" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_capacity_postings_created" ON "carrier_capacity_postings"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_capacity_postings_search" ON "carrier_capacity_postings" USING GIN ("search_vector");

--bun:split
COMMENT ON TABLE "carrier_capacity_postings" IS 'Trucks a carrier says it has available: origin, optional destination and equipment, the window, how many trucks and the rate it asks';

--bun:split
COMMENT ON COLUMN "carrier_capacity_postings"."origin_radius_miles" IS 'How far from the origin location the carrier will pick up; only with an origin location';

--bun:split
COMMENT ON COLUMN "carrier_capacity_postings"."rate" IS 'Asked rate: per loaded mile for PerMile, the whole move for Flat; null when the carrier did not quote';

--bun:split
SELECT trenova_rls.reconcile();
