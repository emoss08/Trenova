ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "found_tools" JSONB;

COMMENT ON COLUMN "assistant_messages"."found_tools" IS 'The tools a find_tools call found, set only on that call''s result, so a later turn reloads the same tools without searching again';
