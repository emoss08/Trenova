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

CREATE TABLE
    IF NOT EXISTS "location_categories"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "name"             VARCHAR(50) NOT NULL,
    "description"      TEXT,
    "color"            VARCHAR(10),
    "version"          BIGINT      NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "location_categories_name_organization_id_unq" ON "location_categories" (LOWER("name"), organization_id);
CREATE INDEX idx_location_categories_name ON location_categories (name);
CREATE INDEX idx_location_categories_org_bu ON location_categories (organization_id, business_unit_id);
CREATE INDEX idx_location_categories_description ON location_categories USING GIN (description gin_trgm_ops);
CREATE INDEX idx_location_categories_created_at ON location_categories (created_at);

-- bun:split

COMMENT ON COLUMN location_categories.id IS 'Unique identifier for the location category, generated as a UUID';
COMMENT ON COLUMN location_categories.business_unit_id IS 'Foreign key referencing the business unit that this location category belongs to';
COMMENT ON COLUMN location_categories.organization_id IS 'Foreign key referencing the organization that this location category belongs to';
COMMENT ON COLUMN location_categories.name IS 'The name of the location category';
COMMENT ON COLUMN location_categories.description IS 'A detailed description of the location category';
COMMENT ON COLUMN location_categories.color IS 'The color associated with the location category, represented as a string limited to 10 characters';
COMMENT ON COLUMN location_categories.created_at IS 'Timestamp of when the location category was created, defaults to the current timestamp';
COMMENT ON COLUMN location_categories.updated_at IS 'Timestamp of the last update to the location category, defaults to the current timestamp';
