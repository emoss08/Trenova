SET statement_timeout = 0;

ALTER TABLE "shipment_import_chat_conversations"
    ADD COLUMN IF NOT EXISTS "status_reason" varchar(64) DEFAULT NULL;

ALTER TABLE "shipment_import_chat_turns"
    ADD COLUMN IF NOT EXISTS "result_status" varchar(32) NOT NULL DEFAULT 'Completed',
    ADD COLUMN IF NOT EXISTS "error_message" text DEFAULT NULL;
