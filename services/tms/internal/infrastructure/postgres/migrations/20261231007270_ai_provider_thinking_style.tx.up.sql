-- A Claude model from Opus 4.6 and Sonnet 4.6 on thinks by effort and refuses
-- a token budget; an older one takes only the budget. The adapter reads which
-- from the model id, and a gateway's alias says nothing, so an operator who
-- knows the model behind it can say which. Auto reads the id. Only the
-- Anthropic Messages protocol has two ways of asking.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "thinking_style" varchar(50) NOT NULL DEFAULT 'Auto';

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_thinking_style" CHECK ("thinking_style" IN ('Auto', 'Effort', 'Budget'));

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_thinking_style_kind" CHECK ("thinking_style" = 'Auto' OR "kind" = 'AnthropicMessages');

--bun:split
COMMENT ON COLUMN "ai_providers"."thinking_style" IS 'How a Claude model is asked to think: Auto reads the model id, Effort asks by effort, Budget with a token budget';
