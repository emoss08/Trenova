DROP INDEX IF EXISTS "idx_accounting_sync_records_external";

--bun:split
DROP TABLE IF EXISTS "accounting_inbound_changes";

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_changes_error_category";

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_inbound_payment_policy";

--bun:split
ALTER TABLE "accounting_connections"
    DROP COLUMN IF EXISTS "inbound_payment_policy",
    DROP COLUMN IF EXISTS "changes_error_message",
    DROP COLUMN IF EXISTS "changes_error_category",
    DROP COLUMN IF EXISTS "changes_read_at",
    DROP COLUMN IF EXISTS "change_cursor";
