DELETE FROM "edi_source_context_fields"
WHERE "id" = 'ediscf_x12_214_event_time_code';

--bun:split
DROP INDEX IF EXISTS "idx_edi_messages_ack";

ALTER TABLE "edi_messages"
    DROP COLUMN IF EXISTS "ack_status",
    DROP COLUMN IF EXISTS "ack_message_id",
    DROP COLUMN IF EXISTS "ack_received_at",
    DROP COLUMN IF EXISTS "ack_last_error";

DROP TYPE IF EXISTS edi_message_ack_status_enum;
