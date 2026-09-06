DROP TABLE IF EXISTS "approval_delegations";
--bun:split
ALTER TABLE "workers"
    DROP CONSTRAINT IF EXISTS "fk_workers_position";
--bun:split
DROP INDEX IF EXISTS "idx_workers_manager";
--bun:split
DROP INDEX IF EXISTS "idx_workers_position";
--bun:split
ALTER TABLE "workers"
    DROP COLUMN IF EXISTS "position_id";
--bun:split
DROP TABLE IF EXISTS "job_positions";
--bun:split
DROP TYPE IF EXISTS "approval_scope_enum";
--bun:split
DROP TYPE IF EXISTS "job_department_enum";
