-- ai_logs.model was a closed Postgres enum, so every model the system could call
-- had to be added by migration. That cannot hold once an organization points
-- Trenova at its own model server: a self-hosted deployment names its models
-- whatever it likes ("llama3.3:70b", "qwen2.5-coder:32b"), and none of those names
-- can be known in advance. The column becomes free text, and provider_kind records
-- which wire protocol produced the row so usage stays attributable.
--
-- The search vector is a generated column that reads "model" through
-- enum_to_text, and Postgres refuses to retype a column a generated column
-- depends on. It is dropped first and rebuilt afterwards over the text column;
-- dropping it also drops its GIN index, which is recreated below.
--
-- This is a type change, not a row update, so the append-only row trigger on
-- ai_logs does not fire.
ALTER TABLE "ai_logs"
    DROP COLUMN IF EXISTS "search_vector";

--bun:split
ALTER TABLE "ai_logs"
    ALTER COLUMN "model" TYPE varchar(200)
    USING "model"::text;

--bun:split
ALTER TABLE "ai_logs"
    ADD COLUMN IF NOT EXISTS "provider_kind" varchar(50) NOT NULL DEFAULT '';

--bun:split
ALTER TABLE "ai_logs"
    ADD COLUMN IF NOT EXISTS "provider_id" varchar(100);

--bun:split
DROP TYPE IF EXISTS "model_enum";

--bun:split
ALTER TABLE "ai_logs"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("prompt", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("operation"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("response", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("model", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_logs_search_vector" ON "ai_logs" USING GIN("search_vector");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_model" ON "ai_logs"("model");

CREATE INDEX IF NOT EXISTS "idx_ai_logs_provider" ON "ai_logs"("provider_id");

COMMENT ON COLUMN "ai_logs"."model" IS 'Free-text model identifier as reported by the provider; not a closed set because self-hosted models are named arbitrarily';

COMMENT ON COLUMN "ai_logs"."provider_kind" IS 'Wire protocol that served the call (AnthropicMessages, OpenAIResponses, OpenAIChat)';
