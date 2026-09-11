--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

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
