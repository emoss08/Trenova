-- Rebuilding the enum from the values actually present, unioned with the six the
-- type originally carried, keeps the rollback lossless. Reinstating only the
-- original six would fail against any row written while the column was free text.
DO $$
DECLARE
    enum_values text;
BEGIN
    SELECT string_agg(DISTINCT quote_literal(value), ', ')
        INTO enum_values
    FROM (
        SELECT unnest(ARRAY[
            'gpt-5-nano',
            'gpt-5-nano-2025-08-07',
            'gpt-5-mini',
            'gpt-5-mini-2025-08-07',
            'omni-moderation-latest',
            'claude-opus-5'
        ]) AS value
        UNION
        SELECT "model" FROM "ai_logs" WHERE "model" IS NOT NULL AND "model" <> ''
    ) AS present;

    EXECUTE format('CREATE TYPE "model_enum" AS ENUM(%s)', enum_values);
END
$$;

--bun:split
DROP INDEX IF EXISTS "idx_ai_logs_model";

DROP INDEX IF EXISTS "idx_ai_logs_provider";

--bun:split
-- The generated search vector depends on "model" and must be rebuilt around
-- the retyped column, as the forward migration did.
ALTER TABLE "ai_logs"
    DROP COLUMN IF EXISTS "search_vector";

--bun:split
ALTER TABLE "ai_logs"
    ALTER COLUMN "model" TYPE model_enum
    USING "model"::model_enum;

--bun:split
ALTER TABLE "ai_logs"
    DROP COLUMN IF EXISTS "provider_kind";

ALTER TABLE "ai_logs"
    DROP COLUMN IF EXISTS "provider_id";

--bun:split
ALTER TABLE "ai_logs"
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('english', COALESCE("prompt", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("operation"), '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE("response", '')), 'B') ||
        setweight(immutable_to_tsvector('english', COALESCE(enum_to_text("model"), '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_logs_search_vector" ON "ai_logs" USING GIN("search_vector");
