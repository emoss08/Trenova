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
