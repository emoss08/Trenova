--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "trailers"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Relationship identifiers (Non-Primary-Keys)
    "equipment_type_id" varchar(100) NOT NULL,
    "equipment_manufacturer_id" varchar(100) NOT NULL,
    "registration_state_id" varchar(100),
    "fleet_code_id" varchar(100),
    -- Core fields
    "status" equipment_status_enum NOT NULL DEFAULT 'Available',
    "code" varchar(50) NOT NULL,
    "model" varchar(50),
    "make" varchar(50),
    "year" int,
    "license_plate_number" varchar(50),
    "vin" vin_code_optional,
    "registration_number" varchar(50),
    "max_load_weight" int,
    "last_inspection_date" bigint,
    "registration_expiry" bigint,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_trailers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_trailers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_equipment_type" FOREIGN KEY ("equipment_type_id", "business_unit_id", "organization_id") REFERENCES "equipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_equipment_manufacturer" FOREIGN KEY ("equipment_manufacturer_id", "business_unit_id", "organization_id") REFERENCES "equipment_manufacturers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_trailers_registration_state" FOREIGN KEY ("registration_state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_trailers_fleet_code" FOREIGN KEY ("fleet_code_id", "business_unit_id", "organization_id") REFERENCES "fleet_codes"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Create unique index for code and organization_id
CREATE UNIQUE INDEX "idx_trailers_code" ON "trailers"(lower("code"), "organization_id");

-- Create index for status
CREATE INDEX "idx_trailers_status" ON "trailers"("status");

CREATE INDEX "idx_trailers_business_unit_organization" ON "trailers"("business_unit_id", "organization_id");

CREATE INDEX "idx_trailers_created_updated" ON "trailers"("created_at", "updated_at");

COMMENT ON TABLE "trailers" IS 'Stores information about trailers';

--bun:split
ALTER TABLE "trailers"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_trailers_search ON trailers USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION trailers_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.vin, '')), 'B') || setweight(to_tsvector('simple', COALESCE(NEW.license_plate_number, '')), 'C') || setweight(to_tsvector('simple', COALESCE(NEW.registration_number, '')), 'D');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS trailers_search_vector_trigger ON trailers;

--bun:split
CREATE TRIGGER trailers_search_vector_trigger
    BEFORE INSERT OR UPDATE ON trailers
    FOR EACH ROW
    EXECUTE FUNCTION trailers_search_vector_update();

--bun:split
ALTER TABLE trailers
    ALTER COLUMN status SET STATISTICS 1000;

--bun:split
ALTER TABLE trailers
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE trailers
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_trailers_trgm_code ON trailers USING gin(code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_trailers_trgm_vin ON trailers USING gin(vin gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_trailers_trgm_license_plate_number ON trailers USING gin(license_plate_number gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_trailers_trgm_registration_number ON trailers USING gin(registration_number gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_trailers_trgm_code_vin_license_plate_number_registration_number ON trailers USING gin((code || ' ' || vin || ' ' || license_plate_number || ' ' || registration_number) gin_trgm_ops);

