ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract'));

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_subject_type";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_subject_type" CHECK ("subject_type" IN ('Document', 'FormulaSchema'));
