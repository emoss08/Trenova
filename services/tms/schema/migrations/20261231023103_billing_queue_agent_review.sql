ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "agent_verdict" varchar(20),
    ADD COLUMN IF NOT EXISTS "agent_summary" text,
    ADD COLUMN IF NOT EXISTS "agent_reviewed_at" bigint,
    ADD COLUMN IF NOT EXISTS "agent_review_hash" varchar(64);

ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "ck_billing_queue_items_agent_verdict" CHECK ("agent_verdict" IS NULL OR "agent_verdict" IN ('Agrees', 'Concern'));

ALTER TABLE "billing_controls"
    ADD COLUMN IF NOT EXISTS "queue_auto_approve_mode" varchar(30) NOT NULL DEFAULT 'Off',
    ADD COLUMN IF NOT EXISTS "queue_auto_approve_max_amount" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "queue_agent_review_enabled" boolean NOT NULL DEFAULT TRUE;

ALTER TABLE "billing_controls"
    ADD CONSTRAINT "ck_billing_controls_queue_auto_approve_mode" CHECK ("queue_auto_approve_mode" IN ('Off', 'WhenChecksClear', 'WhenAgentAgrees')),
    ADD CONSTRAINT "ck_billing_controls_queue_auto_approve_max_amount" CHECK ("queue_auto_approve_max_amount" IS NULL OR "queue_auto_approve_max_amount" > 0);

CREATE TABLE IF NOT EXISTS "billing_queue_briefs" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "text" text NOT NULL,
    "input_hash" varchar(64) NOT NULL,
    "generated_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_billing_queue_briefs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_billing_queue_briefs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_briefs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_queue_briefs_tenant" ON "billing_queue_briefs" ("organization_id", "business_unit_id");

ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS "delivery_facts" jsonb;

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'FactoringRemittanceExtract', 'AgentReflection', 'AgentDrafting', 'AgentProfile', 'AgentRecommendation', 'ConversationChapters', 'BillingQueueReview', 'BillingQueueBrief', 'BillingQueueSearch'));
