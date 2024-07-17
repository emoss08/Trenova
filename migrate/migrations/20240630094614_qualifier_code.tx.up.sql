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
    IF NOT EXISTS "qualifier_codes"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "code"             VARCHAR(10) NOT NULL,
    "description"      TEXT,
    "version"          BIGINT      NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "qualifier_codes_code_organization_id_unq" ON "qualifier_codes" (LOWER("code"), organization_id);
CREATE INDEX idx_qualifier_codes_code ON qualifier_codes (code);
CREATE INDEX idx_qualifier_codes_org_bu ON qualifier_codes (organization_id, business_unit_id);
CREATE INDEX idx_qualifier_codes_description ON qualifier_codes USING GIN (description gin_trgm_ops);
CREATE INDEX idx_qualifier_codes_created_at ON qualifier_codes (created_at);

--bun:split

COMMENT ON COLUMN qualifier_codes.id IS 'Unique identifier for the qualifier code, generated as a UUID';
COMMENT ON COLUMN qualifier_codes.business_unit_id IS 'Foreign key referencing the business unit that this qualifier code belongs to';
COMMENT ON COLUMN qualifier_codes.organization_id IS 'Foreign key referencing the organization that this qualifier code belongs to';
COMMENT ON COLUMN qualifier_codes.status IS 'The current status of the qualifier code, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN qualifier_codes.code IS 'A short, unique code for identifying the qualifier code, limited to 10 characters';
COMMENT ON COLUMN qualifier_codes.description IS 'A detailed description of the qualifier code';
COMMENT ON COLUMN qualifier_codes.created_at IS 'Timestamp of when the qualifier code was created, defaults to the current timestamp';
COMMENT ON COLUMN qualifier_codes.updated_at IS 'Timestamp of the last update to the qualifier code, defaults to the current timestamp';
