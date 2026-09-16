-- A thread belongs to the person who opened it, not to the organization at
-- large. A conversation reveals what someone was investigating and what the
-- assistant showed them, which can include customer rates and worker records.
CREATE TABLE IF NOT EXISTS "assistant_threads"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "title" varchar(200),
    "status" varchar(50) NOT NULL DEFAULT 'Active',
    "last_message_at" bigint NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_assistant_threads" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_threads_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_threads_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_threads_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_threads_status" CHECK ("status" IN ('Active', 'Archived'))
);

--bun:split
-- The thread list is always "my threads, newest first".
CREATE INDEX IF NOT EXISTS "idx_assistant_threads_user" ON "assistant_threads"("organization_id", "business_unit_id", "user_id", "last_message_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_assistant_threads_agent" ON "assistant_threads"("agent_definition_id");

--bun:split
-- Tool traffic is persisted alongside the prose rather than discarded after the
-- loop: "which records did the assistant read before saying that" is the first
-- question anyone reviewing an answer asks. The scope columns record the guard's
-- verdict for the same reason — a refusal is evidence about how the assistant is
-- being used, and only useful if it outlives the request.
CREATE TABLE IF NOT EXISTS "assistant_messages"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "thread_id" varchar(100) NOT NULL,
    "sequence" integer NOT NULL,
    "role" varchar(50) NOT NULL,
    "content" text,
    "tool_calls" jsonb,
    "tool_call_id" varchar(200),
    "tool_name" varchar(200),
    "tool_failed" boolean NOT NULL DEFAULT FALSE,
    "scope_stage" varchar(50),
    "scope_category" varchar(50),
    "scope_reason" varchar(50),
    "refused" boolean NOT NULL DEFAULT FALSE,
    "model" varchar(200),
    "provider_id" varchar(100),
    "input_tokens" integer NOT NULL DEFAULT 0,
    "output_tokens" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_assistant_messages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_messages_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_messages_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_messages_role" CHECK ("role" IN ('User', 'Assistant', 'Tool')),
    CONSTRAINT "ck_assistant_messages_sequence" CHECK ("sequence" >= 0)
);

--bun:split
-- Sequence orders a thread without relying on a timestamp, which collides inside
-- a fast tool loop; the unique index is what keeps that ordering trustworthy.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_messages_sequence" ON "assistant_messages"("thread_id", "sequence");

CREATE INDEX IF NOT EXISTS "idx_assistant_messages_thread" ON "assistant_messages"("organization_id", "business_unit_id", "thread_id", "sequence");

-- Refusals are queried on their own when reviewing how the assistant is used, so
-- they get a partial index rather than a scan over every message.
CREATE INDEX IF NOT EXISTS "idx_assistant_messages_refused" ON "assistant_messages"("organization_id", "business_unit_id", "created_at") WHERE "refused";

--bun:split
COMMENT ON TABLE "assistant_threads" IS 'Per-user conversations with a configured agent';

COMMENT ON TABLE "assistant_messages" IS 'Conversation turns including tool traffic and scope-guard verdicts';

COMMENT ON COLUMN "assistant_messages"."scope_stage" IS 'Which guard layer decided this turn: Deterministic, Classifier, Output, Unavailable, or Allowed';

COMMENT ON COLUMN "assistant_messages"."tool_calls" IS 'Normalized tool requests, stored independently of any provider wire format';
