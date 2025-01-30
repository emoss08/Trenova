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
    "vin" varchar(50),
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
ALTER TABLE "shipment_moves"
    ADD COLUMN "trailer_id" varchar(100) NOT NULL;

ALTER TABLE "shipment_moves"
    ADD CONSTRAINT "fk_shipment_moves_trailer" FOREIGN KEY ("trailer_id", "business_unit_id", "organization_id") REFERENCES "trailers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

