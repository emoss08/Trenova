-- A memory's words alone, ignoring case, spacing and punctuation: "Quote in dollars." and
-- "quote in dollars" are the same memory. Saving one looks for a kept memory with the same
-- key for the same readers, and the key makes that lookup an index read.
ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "content_key" text GENERATED ALWAYS AS (
        btrim(regexp_replace(lower("content"), '[^[:alnum:]]+', ' ', 'g'))
    ) STORED;

COMMENT ON COLUMN "agent_memories"."content_key" IS 'The memory''s words alone, lower-cased with punctuation and spacing folded, so the same memory is never kept twice for the same readers';

-- Checking a new memory against the ones already kept is a model call of its own, attributed on
-- its own line of AI usage.
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentWakeUp', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'FactoringRemittanceExtract', 'AgentReflection', 'AgentMemoryConsolidation', 'AgentDrafting', 'AgentProfile', 'AgentRecommendation', 'ConversationChapters', 'BillingQueueReview', 'BillingQueueBrief', 'BillingQueueSearch', 'PaperworkReading', 'InboundMail', 'PaletteAnswer', 'InsightNarration'));
