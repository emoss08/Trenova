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
    IF NOT EXISTS "tags" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        "status" status_enum NOT NULL DEFAULT 'Active',
        "name" VARCHAR(50) NOT NULL,
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
CREATE UNIQUE INDEX IF NOT EXISTS "tags_name_organization_id_unq" ON "tags" (LOWER("name"), organization_id);

CREATE INDEX idx_tags_name ON tags (name);

CREATE INDEX idx_tags_org_bu ON tags (organization_id, business_unit_id);

CREATE INDEX idx_tags_description ON tags USING GIN (description gin_trgm_ops);

CREATE INDEX idx_tags_created_at ON tags (created_at);

--bun:split
CREATE TABLE
    IF NOT EXISTS "general_ledger_account_tags" (
        "general_ledger_account_id" uuid NOT NULL,
        "tag_id" uuid NOT NULL,
        PRIMARY KEY ("general_ledger_account_id", "tag_id"),
        FOREIGN KEY ("general_ledger_account_id") REFERENCES general_ledger_accounts ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        FOREIGN KEY ("tag_id") REFERENCES tags ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );