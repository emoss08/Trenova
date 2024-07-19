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
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'severity_enum') THEN
            CREATE TYPE severity_enum AS ENUM ('High', 'Medium', 'Low');
        END IF;
    END
$$;

--bun:split

CREATE TABLE
    IF NOT EXISTS "comment_types"
(
    "created_at"       TIMESTAMPTZ   NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ   NOT NULL DEFAULT current_timestamp,
    "id"               uuid          NOT NULL DEFAULT uuid_generate_v4(),
    "status"           status_enum   NOT NULL DEFAULT 'Active',
    "name"             VARCHAR(20)   NOT NULL,
    "severity"         severity_enum NOT NULL DEFAULT 'Low',
    "description"      TEXT          NOT NULL,
    "version"          BIGINT      NOT NULL,
    "business_unit_id" uuid          NOT NULL,
    "organization_id"  uuid          NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "comment_type_name_organization_id_unq" ON "comment_types" (LOWER("name"), organization_id);
CREATE INDEX idx_comment_types_name ON comment_types (name);
CREATE INDEX idx_comment_types_org_bu ON comment_types (organization_id, business_unit_id);
CREATE INDEX idx_comment_types_description ON comment_types USING GIN (description gin_trgm_ops);
CREATE INDEX idx_comment_types_created_at ON comment_types(created_at);

--bun:split

COMMENT ON COLUMN comment_types.id IS 'Unique identifier for the comment type, generated as a UUID';
COMMENT ON COLUMN comment_types.business_unit_id IS 'Foreign key referencing the business unit that this comment type belongs to';
COMMENT ON COLUMN comment_types.organization_id IS 'Foreign key referencing the organization that this comment type belongs to';
COMMENT ON COLUMN comment_types.status IS 'The current status of the comment type, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN comment_types.name IS 'A short, unique name for identifying the comment type, limited to 20 characters';
COMMENT ON COLUMN comment_types.severity IS 'The severity of the comment type, using the severity_enum (e.g., High, Medium, Low)';
COMMENT ON COLUMN comment_types.description IS 'A detailed description of the comment type';
COMMENT ON COLUMN comment_types.created_at IS 'Timestamp of when the comment type was created, defaults to the current timestamp';
COMMENT ON COLUMN comment_types.updated_at IS 'Timestamp of the last update to the comment type, defaults to the current timestamp';