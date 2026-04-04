SET statement_timeout = 0;

ALTER TABLE "shipment_import_chat_turns"
    DROP COLUMN IF EXISTS "error_message",
    DROP COLUMN IF EXISTS "result_status";

ALTER TABLE "shipment_import_chat_conversations"
    DROP COLUMN IF EXISTS "status_reason";
