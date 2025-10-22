SET statement_timeout = 0;

-- Drop search vector trigger
DROP TRIGGER IF EXISTS sso_configs_search_vector_trigger ON sso_configs;

--bun:split
-- Drop search vector function
DROP FUNCTION IF EXISTS sso_configs_update_search_vector();

--bun:split
-- Drop update timestamps trigger
DROP TRIGGER IF EXISTS sso_configs_update_trigger ON sso_configs;

--bun:split
-- Drop update timestamps function
DROP FUNCTION IF EXISTS sso_configs_update_timestamps();

--bun:split
-- Drop the SSO configs table
DROP TABLE IF EXISTS "sso_configs" CASCADE;

--bun:split
-- Drop SSO provider enum type
DROP TYPE IF EXISTS sso_provider_enum;

--bun:split
-- Drop SSO protocol enum type
DROP TYPE IF EXISTS sso_protocol_enum;

