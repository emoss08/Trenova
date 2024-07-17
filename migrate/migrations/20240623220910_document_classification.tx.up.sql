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

CREATE TABLE
    IF NOT EXISTS "document_classifications" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        "status" status_enum NOT NULL DEFAULT 'Active',
        "code" VARCHAR(10) NOT NULL,
        "description" TEXT,
        "color" VARCHAR(10),
        "version" BIGINT NOT NULL,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        PRIMARY KEY ("id"),
        FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "document_classifications_code_organization_id_unq" ON "document_classifications" (LOWER("code"), organization_id);

CREATE INDEX idx_document_classifications_code ON document_classifications (code);

CREATE INDEX idx_document_classifications_org_bu ON document_classifications (organization_id, business_unit_id);

CREATE INDEX idx_document_classifications_description ON document_classifications USING GIN (description gin_trgm_ops);

CREATE INDEX idx_document_classifications_created_at ON document_classifications (created_at);

--bun:split
COMMENT ON COLUMN document_classifications.id IS 'Unique identifier for the document classification, generated as a UUID';

COMMENT ON COLUMN document_classifications.business_unit_id IS 'Foreign key referencing the business unit that this document classification belongs to';

COMMENT ON COLUMN document_classifications.organization_id IS 'Foreign key referencing the organization that this document classification belongs to';

COMMENT ON COLUMN document_classifications.status IS 'The current status of the document classification, using the status_enum (e.g., Active, Inactive)';

COMMENT ON COLUMN document_classifications.code IS 'A short, unique code for identifying the document classification, limited to 10 characters';

COMMENT ON COLUMN document_classifications.description IS 'A detailed description of the document classification';

COMMENT ON COLUMN document_classifications.color IS 'A color code for the document classification';

COMMENT ON COLUMN document_classifications.created_at IS 'Timestamp of when the document classification was created, defaults to the current timestamp';

COMMENT ON COLUMN document_classifications.updated_at IS 'Timestamp of the last update to the document classification, defaults to the current timestamp';