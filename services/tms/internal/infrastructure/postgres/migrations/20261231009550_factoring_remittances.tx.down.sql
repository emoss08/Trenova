--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DELETE FROM "ai_usage_records" WHERE "feature" = 'FactoringRemittanceExtract';

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'AgentReflection', 'AgentDrafting', 'AgentProfile'));

--bun:split
UPDATE "triage_rules" SET "classification" = 'Other' WHERE "classification" = 'FactorRemittance';

--bun:split
ALTER TABLE "triage_rules"
    DROP CONSTRAINT IF EXISTS "ck_triage_rules_classification";

--bun:split
ALTER TABLE "triage_rules"
    ADD CONSTRAINT "ck_triage_rules_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other'));

--bun:split
UPDATE "inbound_messages" SET "classification" = 'Other' WHERE "classification" = 'FactorRemittance';

--bun:split
ALTER TABLE "inbound_messages"
    DROP CONSTRAINT IF EXISTS "ck_inbound_messages_classification";

--bun:split
ALTER TABLE "inbound_messages"
    ADD CONSTRAINT "ck_inbound_messages_classification" CHECK ("classification" IS NULL OR "classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other'));

--bun:split
DROP INDEX IF EXISTS "idx_factoring_reserve_entries_remittance_line";

--bun:split
ALTER TABLE "factoring_reserve_entries"
    DROP COLUMN IF EXISTS "remittance_line_id";

--bun:split
DROP TABLE IF EXISTS "factoring_remittance_profiles";

--bun:split
DROP TABLE IF EXISTS "factoring_remittance_lines";

--bun:split
DROP TABLE IF EXISTS "factoring_remittances";
