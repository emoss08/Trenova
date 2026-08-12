-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260404090000_add_shipment_import_chat_transcripts.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_import_chat_conversations"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "external_conversation_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "turn_count" INTEGER NOT NULL DEFAULT 0,
    "last_message_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_import_chat_conversations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_shipment_import_chat_conversations_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_conversations_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_conversations_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_conversations_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_shipment_import_chat_conversations_active_document"
    ON "shipment_import_chat_conversations" ("organization_id", "business_unit_id", "document_id")WHERE "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_import_chat_conversations_document"
    ON "shipment_import_chat_conversations" ("organization_id", "business_unit_id", "document_id", "updated_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "shipment_import_chat_turns"(
    "id" TEXT NOT NULL,
    "conversation_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "turn_index" INTEGER NOT NULL,
    "user_message" TEXT NOT NULL,
    "assistant_message" TEXT NOT NULL,
    "request_conversation_id" TEXT,
    "response_conversation_id" TEXT,
    "model" TEXT,
    "context_json" TEXT NOT NULL DEFAULT '{}',
    "suggestions_json" TEXT NOT NULL DEFAULT '[]',
    "tool_calls_json" TEXT NOT NULL DEFAULT '[]',
    "actions_json" TEXT NOT NULL DEFAULT '[]',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_import_chat_turns" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_shipment_import_chat_turns_conversation_turn" UNIQUE ("conversation_id", "organization_id", "business_unit_id", "turn_index"),
    CONSTRAINT "fk_shipment_import_chat_turns_conversation" FOREIGN KEY ("conversation_id", "business_unit_id", "organization_id") REFERENCES "shipment_import_chat_conversations"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_turns_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_import_chat_turns_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_import_chat_turns_conversation_created"
    ON "shipment_import_chat_turns" ("organization_id", "business_unit_id", "conversation_id", "turn_index", "created_at");
