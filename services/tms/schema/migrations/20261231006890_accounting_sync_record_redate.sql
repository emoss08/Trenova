ALTER TABLE "accounting_sync_records"
    ADD COLUMN IF NOT EXISTS "redated_to" bigint,
    ADD COLUMN IF NOT EXISTS "redated_by_id" varchar(100);

ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "fk_accounting_sync_records_redated_by" FOREIGN KEY ("redated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;
