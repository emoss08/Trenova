ALTER TABLE "data_retention"
    ADD COLUMN IF NOT EXISTS "edi_inbound_file_retention_period" integer NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "data_retention"
    ADD COLUMN IF NOT EXISTS "edi_message_retention_period" integer NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "edi_inbound_files"
    ADD COLUMN IF NOT EXISTS "raw_purged_at" bigint;

--bun:split
ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "raw_purged_at" bigint;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_inbound_files_raw_retention" ON "edi_inbound_files"("organization_id", "received_at")
WHERE
    "raw_purged_at" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_messages_raw_retention" ON "edi_messages"("organization_id", "created_at")
WHERE
    "raw_purged_at" IS NULL;
