ALTER TABLE "document_controls"
    ADD COLUMN IF NOT EXISTS "enable_ai_assisted_classification" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "enable_ai_assisted_extraction" BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TYPE "operation_enum" ADD VALUE IF NOT EXISTS 'DocumentIntelligenceRoute';
ALTER TYPE "operation_enum" ADD VALUE IF NOT EXISTS 'DocumentIntelligenceExtract';

ALTER TYPE "model_enum" ADD VALUE IF NOT EXISTS 'gpt-5-mini';
ALTER TYPE "model_enum" ADD VALUE IF NOT EXISTS 'gpt-5-mini-2025-08-07';
