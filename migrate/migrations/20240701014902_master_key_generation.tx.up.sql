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
    IF NOT EXISTS "master_key_generations"
(
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    UNIQUE ("organization_id")
);

-- ================================================
-- bun:split

CREATE TABLE
    IF NOT EXISTS "worker_master_key_generations"
(
    "id"            uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "master_key_id" uuid         NOT NULL,
    "pattern"       VARCHAR(255) NOT NULL,
    "created_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- ================================================
-- bun:split

CREATE TABLE
    IF NOT EXISTS "location_master_key_generations"
(
    "id"            uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "master_key_id" uuid         NOT NULL,
    "pattern"       VARCHAR(255) NOT NULL,
    "created_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- ================================================
-- bun:split

CREATE TABLE
    IF NOT EXISTS "customer_master_key_generations"
(
    "id"            uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "master_key_id" uuid         NOT NULL,
    "pattern"       VARCHAR(255) NOT NULL,
    "created_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
