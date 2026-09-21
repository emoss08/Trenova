-- A model that reasons before it answers can be asked to, and what it thought
-- can be kept. Off is the default because the reasoning parameter is refused
-- by models without it; an operator turns it on for a model they know reasons.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "reasoning_effort" varchar(50) NOT NULL DEFAULT 'Off';

--bun:split
ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_reasoning_effort" CHECK ("reasoning_effort" IN ('Off', 'Low', 'Medium', 'High'));

--bun:split
-- The thinking behind an assistant turn: the readable part for the person,
-- and the provider's signed or encrypted part so the next call can continue
-- the same line of thought. Null when the model did not reason out loud.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "reasoning" jsonb;

COMMENT ON COLUMN "ai_providers"."reasoning_effort" IS 'How hard the model is asked to think before answering; Off sends no reasoning parameter';

COMMENT ON COLUMN "assistant_messages"."reasoning" IS 'What the model thought before this turn, plus the provider-opaque state needed to continue it';
