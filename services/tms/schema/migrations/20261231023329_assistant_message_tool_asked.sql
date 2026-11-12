ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "tool_asked" TEXT;

COMMENT ON COLUMN "assistant_messages"."tool_asked" IS 'What a read tool call was narrowed by, in words a person reads: each filter''s label and values with every record by its name, never its id; shown under the step in the conversation';
