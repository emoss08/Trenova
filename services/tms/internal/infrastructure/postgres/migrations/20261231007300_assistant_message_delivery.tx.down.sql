ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "truncated",
    DROP COLUMN IF EXISTS "fallback_from";
