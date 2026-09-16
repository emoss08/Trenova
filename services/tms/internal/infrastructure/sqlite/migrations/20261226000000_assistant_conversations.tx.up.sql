-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261226000000_assistant_conversations.tx.up.sql

CREATE TABLE IF NOT EXISTS "assistant_threads"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "agent_definition_id" TEXT NOT NULL,
    "title" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "last_message_at" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_assistant_threads" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_threads_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_threads_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_threads_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_threads_status" CHECK ("status" IN ('Active', 'Archived'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_threads_user" ON "assistant_threads" ("organization_id", "business_unit_id", "user_id", "last_message_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_threads_agent" ON "assistant_threads" ("agent_definition_id");

--bun:split

CREATE TABLE IF NOT EXISTS "assistant_messages"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "thread_id" TEXT NOT NULL,
    "sequence" INTEGER NOT NULL,
    "role" TEXT NOT NULL,
    "content" TEXT,
    "tool_calls" TEXT,
    "tool_call_id" TEXT,
    "tool_name" TEXT,
    "tool_failed" INTEGER NOT NULL DEFAULT 0,
    "scope_stage" TEXT,
    "scope_category" TEXT,
    "scope_reason" TEXT,
    "refused" INTEGER NOT NULL DEFAULT 0,
    "model" TEXT,
    "provider_id" TEXT,
    "input_tokens" INTEGER NOT NULL DEFAULT 0,
    "output_tokens" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_assistant_messages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_messages_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_messages_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_messages_role" CHECK ("role" IN ('User', 'Assistant', 'Tool')),
    CONSTRAINT "ck_assistant_messages_sequence" CHECK ("sequence" >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_messages_sequence" ON "assistant_messages" ("thread_id", "sequence");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_messages_thread" ON "assistant_messages" ("organization_id", "business_unit_id", "thread_id", "sequence");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_messages_refused" ON "assistant_messages" ("organization_id", "business_unit_id", "created_at")WHERE "refused";
