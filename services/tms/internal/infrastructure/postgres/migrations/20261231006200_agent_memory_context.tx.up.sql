-- recall_memory used to match the whole question as one substring, so "Acme
-- dock hours" found nothing that said "Acme's dock closes at four". Each
-- memory now carries its words: the subject's name first, then what it says,
-- then the tool it is about.
--
-- The 'simple' configuration is deliberate. Memories are short and mostly
-- names, codes and numbers (a customer, a dock, a lane, a PO format), which an
-- English stemmer does nothing for and occasionally mangles, and organizations
-- write them in whatever language they work in. 'simple' keeps every word as
-- written, so the prefix fallback ("deliver" finds "delivery") works on the
-- same lexemes the column stores; rewording is the semantic leg's job.
ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce("subject_label", '')), 'A') ||
        setweight(to_tsvector('simple', coalesce("content", '')), 'B') ||
        setweight(to_tsvector('simple', coalesce("tool_name", '')), 'C')
    ) STORED;

COMMENT ON COLUMN "agent_memories"."search_vector" IS 'The memory''s words for recall_memory, in the simple configuration: the subject label weighted A, the content B, the tool name C';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_search"
    ON "agent_memories" USING gin ("search_vector");

--bun:split

-- How much of a prompt an agent's memory may take, in approximate tokens.
-- Null uses the default of 6000.
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "memory_token_budget" integer;

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "chk_agent_definitions_memory_token_budget";

--bun:split

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "chk_agent_definitions_memory_token_budget" CHECK (
        "memory_token_budget" IS NULL OR "memory_token_budget" BETWEEN 1000 AND 16000
    );

COMMENT ON COLUMN "agent_definitions"."memory_token_budget" IS 'Approximate tokens of recorded memory a prompt of this agent may carry, 1000 to 16000; null for the default of 6000';
