DROP TABLE IF EXISTS "accounting_mappings";

--bun:split
DROP TABLE IF EXISTS "accounting_reference_objects";

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_setup_step";

--bun:split
ALTER TABLE "accounting_connections"
    DROP COLUMN IF EXISTS "reference_refresh_error",
    DROP COLUMN IF EXISTS "reference_refreshed_at",
    DROP COLUMN IF EXISTS "reference_refresh_started_at",
    DROP COLUMN IF EXISTS "setup_step";
