CREATE TABLE IF NOT EXISTS "record_case_options"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "record_kind" varchar(60) NOT NULL,
    "record_id" varchar(100) NOT NULL,
    "key" varchar(40) NOT NULL,
    "position" integer NOT NULL DEFAULT 0,
    "kind" varchar(20) NOT NULL DEFAULT 'Plan',
    "label" varchar(120) NOT NULL,
    "outcome" varchar(40),
    "outcome_note" varchar(60),
    "tone" varchar(20) NOT NULL DEFAULT 'Neutral',
    "cost" varchar(80),
    "description" varchar(400),
    "recommended" boolean NOT NULL DEFAULT FALSE,
    "chosen" boolean NOT NULL DEFAULT FALSE,
    "projection" jsonb,
    "agent_definition_id" varchar(100),
    "run_id" varchar(100),
    "draft_run_id" varchar(100),
    "draft_requested_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_record_case_options" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_record_case_options_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_case_options_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_record_case_options_kind" CHECK ("kind" IN ('Plan', 'Leave')),
    CONSTRAINT "ck_record_case_options_tone" CHECK ("tone" IN ('Neutral', 'Good', 'Warning', 'Bad', 'Info'))
);

COMMENT ON TABLE "record_case_options" IS 'What could happen about a record with open signals: the two or three alternatives an agent offered on its case, one recommended, one chosen';

CREATE UNIQUE INDEX IF NOT EXISTS "uq_record_case_options_key" ON "record_case_options"("organization_id", "business_unit_id", "record_id", "key");

CREATE TABLE IF NOT EXISTS "record_case_readings"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "record_kind" varchar(60) NOT NULL,
    "record_id" varchar(100) NOT NULL,
    "fingerprint" varchar(64) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Writing',
    "segments" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "why_it_matters" varchar(240),
    "if_nothing_changes" varchar(240),
    "model_identifier" varchar(200),
    "provider_id" varchar(100),
    "requested_at" bigint NOT NULL,
    "written_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_record_case_readings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_record_case_readings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_record_case_readings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_record_case_readings_status" CHECK ("status" IN ('Writing', 'Ready', 'Unavailable'))
);

COMMENT ON TABLE "record_case_readings" IS 'Nova''s reading of a record''s case: the verdict with its citations and highlights, kept per record and rewritten when the open signals change';

CREATE UNIQUE INDEX IF NOT EXISTS "uq_record_case_readings_record" ON "record_case_readings"("organization_id", "business_unit_id", "record_id");

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentWakeUp', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'FactoringRemittanceExtract', 'AgentReflection', 'AgentMemoryConsolidation', 'AgentDrafting', 'AgentProfile', 'AgentRecommendation', 'ConversationChapters', 'BillingQueueReview', 'BillingQueueBrief', 'BillingQueueSearch', 'PaperworkReading', 'InboundMail', 'PaletteAnswer', 'InsightNarration', 'CaseReading'));
