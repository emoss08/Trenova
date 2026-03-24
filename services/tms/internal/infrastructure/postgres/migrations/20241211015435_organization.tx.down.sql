--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

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

