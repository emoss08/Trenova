-- Copyright (c) 2024 Trenova Technologies, LLC
--
-- Licensed under the Business Source License 1.1 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     https://trenova.app/pricing/
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
--
-- Key Terms:
-- - Non-production use only
-- - Change Date: 2026-11-16
-- - Change License: GNU General Public License v2 or later
--
-- For full license text, see the LICENSE file in the root directory.

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'equipment_class_enum') THEN
            CREATE TYPE equipment_class_enum AS ENUM ('Undefined', 'Car', 'Van', 'Pickup', 'Straight', 'Tractor', 'Trailer', 'Container','Chassis');
        END IF;
    END
$$;

--bun:split

CREATE TABLE
    IF NOT EXISTS "equipment_types"
(
    "id"                uuid                 NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id"  uuid                 NOT NULL,
    "organization_id"   uuid                 NOT NULL,
    "status"            status_enum          NOT NULL DEFAULT 'Active',
    "code"              VARCHAR(10)          NOT NULL,
    "equipment_class"   equipment_class_enum NOT NULL DEFAULT 'Undefined',
    "description"       TEXT,
    "cost_per_mile"     NUMERIC(10, 2),
    "fixed_cost"        NUMERIC(10, 2),
    "variable_cost"     NUMERIC(10, 2),
    "height"            NUMERIC(10, 2),
    "length"            NUMERIC(10, 2),
    "width"             NUMERIC(10, 2),
    "weight"            NUMERIC(10, 2),
    "exempt_from_tolls" BOOLEAN              NOT NULL DEFAULT FALSE,
    "color"             VARCHAR(10),
    "version"           BIGINT               NOT NULL,
    "created_at"        TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    "updated_at"        TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "equipment_types_code_organization_id_unq" ON "equipment_types" (LOWER("code"), organization_id);
CREATE INDEX idx_equipment_types_code ON equipment_types (code);
CREATE INDEX idx_equipment_types_org_bu ON equipment_types (organization_id, business_unit_id);
CREATE INDEX idx_equipment_types_description ON equipment_types USING GIN (description gin_trgm_ops);
CREATE INDEX idx_equipment_types_created_at ON equipment_types (created_at);

--bun:split
COMMENT ON COLUMN equipment_types.id IS 'Unique identifier for the equipment type, generated as a UUID';
COMMENT ON COLUMN equipment_types.business_unit_id IS 'Foreign key referencing the business unit that this equipment type belongs to';
COMMENT ON COLUMN equipment_types.organization_id IS 'Foreign key referencing the organization that this equipment type belongs to';
COMMENT ON COLUMN equipment_types.status IS 'The current status of the equipment type, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN equipment_types.code IS 'A short, unique code for identifying the equipment type, limited to 10 characters';
COMMENT ON COLUMN equipment_types.equipment_class IS 'Classification of the equipment, using the equipment_class_enum (e.g., Car, Van, Trailer)';
COMMENT ON COLUMN equipment_types.description IS 'A detailed description of the equipment type';
COMMENT ON COLUMN equipment_types.cost_per_mile IS 'The operational cost per mile for using this equipment type, represented as a numeric value with two decimal places';
COMMENT ON COLUMN equipment_types.fixed_cost IS 'The fixed cost associated with this equipment type, represented as a numeric value with two decimal places';
COMMENT ON COLUMN equipment_types.variable_cost IS 'The variable cost associated with this equipment type, represented as a numeric value with two decimal places';
COMMENT ON COLUMN equipment_types.height IS 'The height of the equipment in specified units, represented as a numeric value with two decimal places';
COMMENT ON COLUMN equipment_types.length IS 'The length of the equipment in specified units, represented as a numeric value with two decimal places';
COMMENT ON COLUMN equipment_types.width IS 'The width of the equipment in specified units, represented as a numeric value with two decimal places';
COMMENT ON COLUMN equipment_types.weight IS 'The weight of the equipment in specified units, represented as a numeric value with two decimal places';
COMMENT ON COLUMN equipment_types.exempt_from_tolls IS 'A boolean flag indicating whether the equipment is exempt from toll charges';
COMMENT ON COLUMN equipment_types.color IS 'The color of the equipment, represented as a string limited to 10 characters';
COMMENT ON COLUMN equipment_types.created_at IS 'Timestamp of when the equipment type was created, defaults to the current timestamp';
COMMENT ON COLUMN equipment_types.updated_at IS 'Timestamp of the last update to the equipment type, defaults to the current timestamp';
