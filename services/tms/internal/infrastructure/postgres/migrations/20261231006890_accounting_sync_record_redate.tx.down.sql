ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "fk_accounting_sync_records_redated_by";

ALTER TABLE "accounting_sync_records"
    DROP COLUMN IF EXISTS "redated_to",
    DROP COLUMN IF EXISTS "redated_by_id";
