DROP TABLE IF EXISTS settlement_disputes;

--bun:split
DROP TABLE IF EXISTS worker_portal_invitations;

--bun:split
DROP INDEX IF EXISTS uq_workers_org_user;

--bun:split
ALTER TABLE workers
    DROP CONSTRAINT IF EXISTS fk_workers_user;

--bun:split
ALTER TABLE workers
    DROP COLUMN IF EXISTS user_id;
