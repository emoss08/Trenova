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
    IF NOT EXISTS "tags"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "status"           status_enum NOT NULL DEFAULT 'Active',
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

CREATE UNIQUE INDEX IF NOT EXISTS "tags_name_organization_id_unq" ON "tags" (LOWER("name"), organization_id);
CREATE INDEX idx_tags_name ON tags (name);
CREATE INDEX idx_tags_org_bu ON tags (organization_id, business_unit_id);
CREATE INDEX idx_tags_description ON tags USING GIN (description gin_trgm_ops);
CREATE INDEX idx_tags_created_at ON tags (created_at);

--bun:split

CREATE TABLE
    IF NOT EXISTS "general_ledger_account_tags"
(
    "general_ledger_account_id" uuid NOT NULL,
    "tag_id"                    uuid NOT NULL,
    PRIMARY KEY ("general_ledger_account_id", "tag_id"),
    FOREIGN KEY ("general_ledger_account_id") REFERENCES general_ledger_accounts ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("tag_id") REFERENCES tags ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
