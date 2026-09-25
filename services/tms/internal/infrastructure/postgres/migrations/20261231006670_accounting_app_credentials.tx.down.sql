ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_app_environment";

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_app_source";

--bun:split
ALTER TABLE "accounting_connections"
    DROP COLUMN IF EXISTS "app_fingerprint",
    DROP COLUMN IF EXISTS "app_environment",
    DROP COLUMN IF EXISTS "app_source";

--bun:split
DROP TABLE IF EXISTS "accounting_app_credentials";
