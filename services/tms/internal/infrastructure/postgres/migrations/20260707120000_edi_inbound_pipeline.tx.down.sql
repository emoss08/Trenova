DROP INDEX IF EXISTS "idx_edi_load_tender_transfers_inbound_message";

--bun:split
ALTER TABLE "edi_load_tender_transfers"
    DROP COLUMN IF EXISTS "inbound_message_id";

--bun:split
DROP INDEX IF EXISTS "idx_edi_messages_ack_lookup";

--bun:split
DROP INDEX IF EXISTS "idx_edi_messages_inbound_file";

--bun:split
ALTER TABLE "edi_messages"
    DROP COLUMN IF EXISTS "inbound_file_id";

--bun:split
DROP TABLE IF EXISTS "edi_inbound_files";

--bun:split
DROP TYPE IF EXISTS "edi_inbound_file_status_enum";
