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

DROP TYPE IF EXISTS "status_enum" CASCADE;

-- bun:split
DROP TYPE IF EXISTS database_action_enum CASCADE;

-- bun:split
DROP TYPE IF EXISTS delivery_method_enum CASCADE;

-- bun:split
DROP TABLE IF EXISTS "us_states" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "business_units" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "organizations" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "resources" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "permissions" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "roles" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "role_permissions" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "users" CASCADE;

-- bun:split
DROP INDEX IF EXISTS "users_email_unq" CASCADE;

-- bun:split
DROP INDEX IF EXISTS "users_username_organization_id_unq" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "user_favorites" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "user_roles" CASCADE;

-- bun:split
DROP TABLE IF EXISTS "fleet_codes" CASCADE;

-- bun:split
DROP INDEX IF EXISTS "fleet_codes_code_organization_id_unq" CASCADE;