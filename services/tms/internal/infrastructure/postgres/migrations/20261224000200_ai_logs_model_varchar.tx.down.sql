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
ALTER TABLE "ai_logs"
    ALTER COLUMN "model" TYPE model_enum
    USING "model"::model_enum;

--bun:split
ALTER TABLE "ai_logs"
    DROP COLUMN IF EXISTS "provider_kind";

ALTER TABLE "ai_logs"
    DROP COLUMN IF EXISTS "provider_id";
