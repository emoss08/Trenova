SET statement_timeout = 0;

DROP INDEX IF EXISTS "idx_shipment_import_chat_turns_conversation_created";
DROP TABLE IF EXISTS "shipment_import_chat_turns";

DROP INDEX IF EXISTS "idx_shipment_import_chat_conversations_document";
DROP INDEX IF EXISTS "idx_shipment_import_chat_conversations_active_document";
DROP TABLE IF EXISTS "shipment_import_chat_conversations";
