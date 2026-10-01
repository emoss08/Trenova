CREATE TABLE IF NOT EXISTS "assistant_thread_summaries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "thread_id" varchar(100) NOT NULL,
    "from_sequence" integer NOT NULL,
    "through_sequence" integer NOT NULL,
    "content" text NOT NULL,
    "sections" jsonb NOT NULL,
    "trigger" varchar(20) NOT NULL,
    "tainted" boolean NOT NULL DEFAULT FALSE,
    "taint" jsonb,
    "messages_summarized" integer NOT NULL DEFAULT 0,
    "tokens_before" integer NOT NULL DEFAULT 0,
    "tokens_after" integer NOT NULL DEFAULT 0,
    "memory_suggestions" integer NOT NULL DEFAULT 0,
    "provider_id" varchar(100),
    "model" varchar(200),
    "input_tokens" integer NOT NULL DEFAULT 0,
    "output_tokens" integer NOT NULL DEFAULT 0,
    "cost_usd" numeric(14, 6),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_assistant_thread_summaries" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_thread_summaries_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_thread_summaries_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_thread_summaries_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_thread_summaries_trigger" CHECK ("trigger" IN ('Threshold', 'Overflow')),
    CONSTRAINT "ck_assistant_thread_summaries_range" CHECK ("from_sequence" >= 0 AND "through_sequence" >= "from_sequence"),
    CONSTRAINT "ck_assistant_thread_summaries_taint" CHECK ("tainted" = ("taint" IS NOT NULL)),
    CONSTRAINT "ck_assistant_thread_summaries_counts" CHECK ("messages_summarized" >= 0 AND "tokens_before" >= 0 AND "tokens_after" >= 0 AND "memory_suggestions" >= 0 AND "input_tokens" >= 0 AND "output_tokens" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_thread_summaries_through" ON "assistant_thread_summaries"("thread_id", "through_sequence");

CREATE INDEX IF NOT EXISTS "idx_assistant_thread_summaries_thread" ON "assistant_thread_summaries"("organization_id", "business_unit_id", "thread_id", "through_sequence" DESC);

--bun:split
COMMENT ON TABLE "assistant_thread_summaries" IS 'Summaries of the earlier part of a conversation, replayed to the model in place of the messages they cover. The messages themselves are kept and shown';

COMMENT ON COLUMN "assistant_thread_summaries"."through_sequence" IS 'The last message sequence the summary covers; the model is replayed this summary and the messages after it';

COMMENT ON COLUMN "assistant_thread_summaries"."tainted" IS 'Set when the conversation had read content from outside the organization, so the summary is replayed fenced as outside content';
