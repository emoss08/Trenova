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

DROP TABLE IF EXISTS "locations" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "location_comments" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "location_contacts" CASCADE;

-- bun:split
DROP INDEX IF EXISTS "locations_code_organization_id_unq" CASCADE;

DROP INDEX IF EXISTS "idx_location_name" CASCADE;

DROP INDEX IF EXISTS "idx_location_org_bu" CASCADE;

DROP INDEX IF EXISTS "idx_location_description" CASCADE;

DROP INDEX IF EXISTS "idx_location_created_at" CASCADE;

DROP INDEX IF EXISTS "idx_location_comment_comment_type_id" CASCADE;

DROP INDEX IF EXISTS "idx_location_comment_created_at" CASCADE;

DROP INDEX IF EXISTS "idx_location_comment_created_at" CASCADE;

DROP INDEX IF EXISTS "idx_location_contact_name" CASCADE;

DROP INDEX IF EXISTS "idx_location_contact_created_at" CASCADE;