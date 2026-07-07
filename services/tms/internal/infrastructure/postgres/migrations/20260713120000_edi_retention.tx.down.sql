DROP INDEX IF EXISTS "idx_edi_inbound_files_raw_retention";

--bun:split
DROP INDEX IF EXISTS "idx_edi_messages_raw_retention";

--bun:split
ALTER TABLE "edi_inbound_files"
    DROP COLUMN IF EXISTS "raw_purged_at";

--bun:split
ALTER TABLE "edi_messages"
    DROP COLUMN IF EXISTS "raw_purged_at";

--bun:split
ALTER TABLE "data_retention"
    DROP COLUMN IF EXISTS "edi_inbound_file_retention_period";

--bun:split
ALTER TABLE "data_retention"
    DROP COLUMN IF EXISTS "edi_message_retention_period";
