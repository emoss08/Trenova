DROP TABLE IF EXISTS "tractors" CASCADE;

-- bun:split

DROP INDEX IF EXISTS "tractors_code_organization_id_unq" CASCADE;
DROP INDEX IF EXISTS "idx_tractors_code" CASCADE;
DROP INDEX IF EXISTS "idx_tractors_org_bu" CASCADE;
DROP INDEX IF EXISTS "tractors_primary_worker_id_unq" CASCADE;
