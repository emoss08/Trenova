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
    IF NOT EXISTS "master_key_generations" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
        "business_unit_id" uuid NOT NULL,
        "organization_id" uuid NOT NULL,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        PRIMARY KEY ("id"),
        FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
        UNIQUE ("organization_id")
    );

-- ================================================
-- bun:split
CREATE TABLE
    IF NOT EXISTS "worker_master_key_generations" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
        "master_key_id" uuid NOT NULL,
        "pattern" VARCHAR(255) NOT NULL,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        PRIMARY KEY ("id"),
        FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );

-- ================================================
-- bun:split
CREATE TABLE
    IF NOT EXISTS "location_master_key_generations" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
        "master_key_id" uuid NOT NULL,
        "pattern" VARCHAR(255) NOT NULL,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        PRIMARY KEY ("id"),
        FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );

-- ================================================
-- bun:split
CREATE TABLE
    IF NOT EXISTS "customer_master_key_generations" (
        "id" uuid NOT NULL DEFAULT uuid_generate_v4 (),
        "master_key_id" uuid NOT NULL,
        "pattern" VARCHAR(255) NOT NULL,
        "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
        PRIMARY KEY ("id"),
        FOREIGN KEY ("master_key_id") REFERENCES master_key_generations ("id") ON UPDATE NO ACTION ON DELETE CASCADE
    );