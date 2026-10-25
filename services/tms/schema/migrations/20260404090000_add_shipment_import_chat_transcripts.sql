SET statement_timeout = 0;

CREATE TABLE IF NOT EXISTS "shipment_import_chat_conversations"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "document_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "external_conversation_id" varchar(255) DEFAULT NULL,
    "status" varchar(32) NOT NULL DEFAULT 'Active',
    "turn_count" integer NOT NULL DEFAULT 0,
    "last_message_at" bigint DEFAULT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_shipment_import_chat_conversations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_shipment_import_chat_conversations_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_conversations_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_conversations_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_conversations_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_shipment_import_chat_conversations_active_document"
    ON "shipment_import_chat_conversations"("organization_id", "business_unit_id", "document_id")
    WHERE "status" = 'Active';

CREATE INDEX IF NOT EXISTS "idx_shipment_import_chat_conversations_document"
    ON "shipment_import_chat_conversations"("organization_id", "business_unit_id", "document_id", "updated_at" DESC);

CREATE TABLE IF NOT EXISTS "shipment_import_chat_turns"(
    "id" varchar(100) NOT NULL,
    "conversation_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "document_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "turn_index" integer NOT NULL,
    "user_message" text NOT NULL,
    "assistant_message" text NOT NULL,
    "request_conversation_id" varchar(255) DEFAULT NULL,
    "response_conversation_id" varchar(255) DEFAULT NULL,
    "model" varchar(100) DEFAULT NULL,
    "context_json" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "suggestions_json" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "tool_calls_json" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "actions_json" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_shipment_import_chat_turns" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_shipment_import_chat_turns_conversation_turn" UNIQUE ("conversation_id", "organization_id", "business_unit_id", "turn_index"),
    CONSTRAINT "fk_shipment_import_chat_turns_conversation" FOREIGN KEY ("conversation_id", "business_unit_id", "organization_id") REFERENCES "shipment_import_chat_conversations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_turns_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_turns_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_shipment_import_chat_turns_conversation_created"
    ON "shipment_import_chat_turns"("organization_id", "business_unit_id", "conversation_id", "turn_index", "created_at");
