DROP TABLE IF EXISTS "accounting_backfills";

--bun:split
DROP TABLE IF EXISTS "accounting_sync_attempts";

--bun:split
DROP TABLE IF EXISTS "accounting_sync_records";

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "fk_accounting_connections_paused_by",
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_sync_ready",
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_setup_step";

--bun:split
UPDATE "accounting_connections" SET "setup_step" = 'Complete' WHERE "setup_step" = 'StartDate';

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_setup_step" CHECK ("setup_step" IN ('Mappings', 'Complete'));

--bun:split
ALTER TABLE "accounting_connections"
    DROP COLUMN IF EXISTS "paused_reason",
    DROP COLUMN IF EXISTS "paused_by_id",
    DROP COLUMN IF EXISTS "paused_at",
    DROP COLUMN IF EXISTS "auto_sync",
    DROP COLUMN IF EXISTS "sync_enabled_at",
    DROP COLUMN IF EXISTS "sync_start_date";
