ALTER TABLE "assistant_briefings"
    ADD COLUMN IF NOT EXISTS "model_identifier" VARCHAR(255);

ALTER TABLE "assistant_briefings"
    ADD COLUMN IF NOT EXISTS "provider_id" VARCHAR(100);

COMMENT ON COLUMN "assistant_briefings"."model_identifier" IS 'The model whose wording was accepted into the briefing; empty when the page reads in the computed wording';

COMMENT ON COLUMN "assistant_briefings"."provider_id" IS 'The AI provider that served the model whose wording was accepted; empty when no model wording was accepted';
