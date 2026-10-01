ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "tool_verdict" VARCHAR(50);

COMMENT ON COLUMN "assistant_messages"."tool_verdict" IS 'On a tool result: how the runtime judged the call, such as ran, proposed, denied, invalid, duplicate, over_budget or failed; null for a result saved before it was kept';
