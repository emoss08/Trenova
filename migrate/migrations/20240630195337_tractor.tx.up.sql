CREATE TABLE
    IF NOT EXISTS "tractors"
(
    "id"                        uuid                  NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"          uuid                  NOT NULL,
    "organization_id"           uuid                  NOT NULL,
    "code"                      VARCHAR(50)           NOT NULL,
    "status"                    equipment_status_enum NOT NULL DEFAULT 'Available',
    "equipment_type_id"         uuid                  NOT NULL,
    "equipment_manufacturer_id" uuid,
    "model"                     VARCHAR(50),
    "year"                      INTEGER,
    "license_plate_number"      VARCHAR(50),
    "vin"                       VARCHAR(17),
    "state_id"                  uuid,
    "fleet_code_id"             uuid,
    "primary_worker_id"         uuid                  NOT NULL,
    "secondary_worker_id"       uuid,
    "is_leased"                 bool                           DEFAULT false,
    "leased_date"               DATE,
    "version"                   BIGINT                NOT NULL,
    "created_at"                TIMESTAMPTZ           NOT NULL DEFAULT current_timestamp,
    "updated_at"                TIMESTAMPTZ           NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("equipment_type_id") REFERENCES equipment_types ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("equipment_manufacturer_id") REFERENCES equipment_manufacturers ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("state_id") REFERENCES us_states ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("fleet_code_id") REFERENCES fleet_codes ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("primary_worker_id") REFERENCES workers ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("secondary_worker_id") REFERENCES workers ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "primary_secondary_worker_check" CHECK ("primary_worker_id" <> "secondary_worker_id")
);

-- bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "tractors_code_organization_id_unq" ON "tractors" (LOWER("code"), organization_id);
CREATE INDEX idx_tractors_code ON tractors (code);
CREATE INDEX idx_tractors_org_bu ON tractors (organization_id, business_unit_id);
CREATE UNIQUE INDEX IF NOT EXISTS "tractors_primary_worker_id_unq" ON "tractors" (primary_worker_id) WHERE primary_worker_id IS NOT NULL;

--bun:split

COMMENT ON COLUMN tractors.id IS 'Unique identifier for the tractor, generated as a UUID';
COMMENT ON COLUMN tractors.business_unit_id IS 'Foreign key referencing the business unit that this tractor belongs to';
COMMENT ON COLUMN tractors.organization_id IS 'Foreign key referencing the organization that this tractor belongs to';
COMMENT ON COLUMN tractors.code IS 'A short, unique code for identifying the tractor, limited to 50 characters';
COMMENT ON COLUMN tractors.status IS 'The current status of the tractor, using the equipment_status_enum (e.g., Available, In Use, Down)';
COMMENT ON COLUMN tractors.equipment_type_id IS 'Foreign key referencing the equipment type that this tractor belongs to';
COMMENT ON COLUMN tractors.equipment_manufacturer_id IS 'Foreign key referencing the equipment manufacturer that this tractor belongs to';
COMMENT ON COLUMN tractors.model IS 'The model of the tractor, limited to 50 characters';
COMMENT ON COLUMN tractors.year IS 'The year the tractor was manufactured';
COMMENT ON COLUMN tractors.license_plate_number IS 'The license plate number of the tractor, limited to 50 characters';
COMMENT ON COLUMN tractors.vin IS 'The vehicle identification number (VIN) of the tractor, limited to 17 characters';
COMMENT ON COLUMN tractors.state_id IS 'Foreign key referencing the state that the tractor is registered in';
COMMENT ON COLUMN tractors.fleet_code_id IS 'Foreign key referencing the fleet code that the tractor belongs to';
COMMENT ON COLUMN tractors.primary_worker_id IS 'Foreign key referencing the primary worker assigned to the tractor';
COMMENT ON COLUMN tractors.secondary_worker_id IS 'Foreign key referencing the secondary worker assigned to the tractor';
COMMENT ON COLUMN tractors.is_leased IS 'Flag indicating if the tractor is leased';
COMMENT ON COLUMN tractors.leased_date IS 'The date the tractor was leased';
COMMENT ON COLUMN tractors.created_at IS 'Timestamp of when the tractor was created, defaults to the current timestamp';
COMMENT ON COLUMN tractors.updated_at IS 'Timestamp of the last update to the tractor, defaults to the current timestamp';
