-- A conversation can park on a record changing (RecordChanged: a person fixing what
-- blocked a write) and on a report run it started finishing (ReportRun).
ALTER TABLE "agent_waits"
    DROP CONSTRAINT IF EXISTS "ck_agent_waits_kind";

ALTER TABLE "agent_waits"
    ADD CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow', 'PortalRun', 'DocumentRead', 'WebWatch', 'RecordChanged', 'ReportRun'));

-- A turn a wait starts is attributed on its own line of AI usage.
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentWakeUp', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'FactoringRemittanceExtract', 'AgentReflection', 'AgentDrafting', 'AgentProfile', 'AgentRecommendation', 'ConversationChapters', 'BillingQueueReview', 'BillingQueueBrief', 'BillingQueueSearch', 'PaperworkReading', 'InboundMail', 'PaletteAnswer', 'InsightNarration'));
