ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "tool_summary" TEXT;

COMMENT ON COLUMN "assistant_messages"."tool_summary" IS 'One-line label for a tool result, such as the page opened or the record found, shown as the step in the conversation';
