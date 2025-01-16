DROP TABLE IF EXISTS "organizations" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_organizations_name" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_organizations_scac_code" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_organizations_dot_number" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_organizations_created_at" CASCADE;

