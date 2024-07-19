-- COPYRIGHT(c) 2024 Trenova
--
-- This file is part of Trenova.
--
-- The Trenova software is licensed under the Business Source License 1.1. You are granted the right
-- to copy, modify, and redistribute the software, but only for non-production use or with a total
-- of less than three server instances. Starting from the Change Date (November 16, 2026), the
-- software will be made available under version 2 or later of the GNU General Public License.
-- If you use the software in violation of this license, your rights under the license will be
-- terminated automatically. The software is provided "as is," and the Licensor disclaims all
-- warranties and conditions. If you use this license's text or the "Business Source License" name
-- and trademark, you must comply with the Licensor's covenants, which include specifying the
-- Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
-- Grant, and not modifying the license in any other way.
DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'equipment_status_enum') THEN CREATE TYPE equipment_status_enum AS ENUM (
            'Available',
            'OutOfService',
            'AtMaintenance',
            'Sold',
            'Lost'
            );

        END IF;

    END
$$;

--bun:split
CREATE TABLE IF NOT EXISTS "trailers"
(
    "id"                           uuid                  NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"             uuid                  NOT NULL,
    "organization_id"              uuid                  NOT NULL,
    "code"                         VARCHAR(50)           NOT NULL,
    "status"                       equipment_status_enum NOT NULL DEFAULT 'Available',
    "equipment_type_id"            uuid                  NOT NULL,
    "equipment_manufacturer_id"    uuid,
    "model"                        VARCHAR(50),
    "year"                         INTEGER,
    "license_plate_number"         VARCHAR(50),
    "vin"                          VARCHAR(17),
    "state_id"                     uuid,
    "fleet_code_id"                uuid,
    "last_inspection_date"         DATE,
    "registration_number"          VARCHAR(50),
    "registration_state_id"        uuid,
    "registration_expiration_date" DATE,
    "version"                      BIGINT                NOT NULL,
    "created_at"                   TIMESTAMPTZ           NOT NULL DEFAULT current_timestamp,
    "updated_at"                   TIMESTAMPTZ           NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("equipment_type_id") REFERENCES equipment_types ("id") ON UPDATE NO ACTION ON DELETE
        SET
        NULL,
    FOREIGN KEY ("equipment_manufacturer_id") REFERENCES equipment_manufacturers ("id") ON UPDATE NO ACTION ON DELETE
        SET
        NULL,
    FOREIGN KEY ("state_id") REFERENCES us_states ("id") ON UPDATE NO ACTION ON DELETE
        SET
        NULL,
    FOREIGN KEY ("fleet_code_id") REFERENCES fleet_codes ("id") ON UPDATE NO ACTION ON DELETE
        SET
        NULL,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "trailers_code_organization_id_unq" ON "trailers" (LOWER("code"), organization_id);

CREATE INDEX idx_trailers_name ON trailers (code);

CREATE INDEX idx_trailers_org_bu ON trailers (organization_id, business_unit_id);

CREATE INDEX idx_trailers_created_at ON trailers (created_at);

--bun:split
COMMENT ON COLUMN trailers.id IS 'Unique identifier for the trailer, generated as a UUID';

COMMENT ON COLUMN trailers.business_unit_id IS 'Foreign key referencing the business unit that this trailer belongs to';

COMMENT ON COLUMN trailers.organization_id IS 'Foreign key referencing the organization that this trailer belongs to';

COMMENT ON COLUMN trailers.code IS 'A short, unique code for identifying the trailer, limited to 50 characters';

COMMENT ON COLUMN trailers.status IS 'The current status of the trailer, using the equipment_status_enum (e.g., Available, OutOfService, AtMaintenance, Sold, Lost)';

COMMENT ON COLUMN trailers.equipment_type_id IS 'Foreign key referencing the equipment type of the trailer';

COMMENT ON COLUMN trailers.equipment_manufacturer_id IS 'Foreign key referencing the manufacturer of the trailer';

COMMENT ON COLUMN trailers.model IS 'The model of the trailer, limited to 50 characters';

COMMENT ON COLUMN trailers.year IS 'The year the trailer was manufactured';

COMMENT ON COLUMN trailers.license_plate_number IS 'The license plate number of the trailer, limited to 50 characters';

COMMENT ON COLUMN trailers.vin IS 'The Vehicle Identification Number (VIN) of the trailer, limited to 17 characters';

COMMENT ON COLUMN trailers.state_id IS 'Foreign key referencing the state of the trailer';

COMMENT ON COLUMN trailers.fleet_code_id IS 'Foreign key referencing the fleet code of the trailer';

COMMENT ON COLUMN trailers.last_inspection_date IS 'The date of the last inspection of the trailer';

COMMENT ON COLUMN trailers.registration_number IS 'The registration number of the trailer, limited to 50 characters';

COMMENT ON COLUMN trailers.registration_state_id IS 'Foreign key referencing the state of registration of the trailer';

COMMENT ON COLUMN trailers.registration_expiration_date IS 'The expiration date of the trailer registration';

COMMENT ON COLUMN trailers.created_at IS 'Timestamp of when the trailer was created, defaults to the current timestamp';

COMMENT ON COLUMN trailers.updated_at IS 'Timestamp of the last update to the trailer, defaults to the current timestamp';