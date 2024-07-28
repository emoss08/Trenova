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
CREATE TABLE IF NOT EXISTS "commodities" (
    "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
    "business_unit_id" uuid NOT NULL,
    "organization_id" uuid NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" VARCHAR(100) NOT NULL,
    "is_hazmat" BOOLEAN NOT NULL DEFAULT FALSE,
    "unit_of_measure" VARCHAR(50),
    "min_temp" INTEGER,
    "max_temp" INTEGER,
    "hazardous_material_id" uuid,
    "version" BIGINT NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("hazardous_material_id") REFERENCES hazardous_materials ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "commodities_name_organization_id_unq" ON "commodities" (LOWER("name"), organization_id);

CREATE INDEX IF NOT EXISTS idx_commodities_name ON commodities (name);

CREATE INDEX IF NOT EXISTS idx_commodities_org_bu ON commodities (organization_id, business_unit_id);

CREATE INDEX IF NOT EXISTS idx_commodities_created_at ON charge_types (created_at);

--bun:split
COMMENT ON COLUMN commodities.id IS 'Unique identifier for the commodity, generated as a UUID';

COMMENT ON COLUMN commodities.business_unit_id IS 'Foreign key referencing the business unit that this commodity belongs to';

COMMENT ON COLUMN commodities.organization_id IS 'Foreign key referencing the organization that this commodity belongs to';

COMMENT ON COLUMN commodities.status IS 'The current status of the commodity, using the status_enum (e.g., Active, Inactive)';

COMMENT ON COLUMN commodities.name IS 'A short, unique name for identifying the commodity, limited to 100 characters';

COMMENT ON COLUMN commodities.is_hazmat IS 'Indicates whether the commodity is hazardous material';

COMMENT ON COLUMN commodities.unit_of_measure IS 'The unit of measure for the commodity';

COMMENT ON COLUMN commodities.min_temp IS 'The minimum temperature for the commodity';

COMMENT ON COLUMN commodities.max_temp IS 'The maximum temperature for the commodity';

COMMENT ON COLUMN commodities.hazardous_material_id IS 'Foreign key referencing the hazardous material that this commodity is associated with';

COMMENT ON COLUMN commodities.created_at IS 'Timestamp of when the commodity was created, defaults to the current timestamp';

COMMENT ON COLUMN commodities.updated_at IS 'Timestamp of the last update to the commodity, defaults to the current timestamp';