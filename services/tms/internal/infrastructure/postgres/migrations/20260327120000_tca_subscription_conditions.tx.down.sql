ALTER TABLE "tca_subscriptions"
    DROP COLUMN IF EXISTS "conditions",
    DROP COLUMN IF EXISTS "condition_match",
    DROP COLUMN IF EXISTS "watched_columns",
    DROP COLUMN IF EXISTS "custom_title",
    DROP COLUMN IF EXISTS "custom_message",
    DROP COLUMN IF EXISTS "topic",
    DROP COLUMN IF EXISTS "priority";
