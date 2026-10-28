CREATE TABLE IF NOT EXISTS "assistant_thread_chapters" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "thread_id" VARCHAR(100) NOT NULL,
    "turn_id" VARCHAR(100) NOT NULL,
    "title" VARCHAR(200) NOT NULL,
    "title_source" VARCHAR(20) NOT NULL,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_assistant_thread_chapters" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_assistant_thread_chapters_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads" ("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_thread_chapters_turn" FOREIGN KEY ("turn_id", "business_unit_id", "organization_id") REFERENCES "assistant_turns" ("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_thread_chapters_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_thread_chapters_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_thread_chapters_title_source" CHECK ("title_source" IN ('Model', 'Question', 'Person', 'Schedule')),
    CONSTRAINT "ck_assistant_thread_chapters_title" CHECK (length(btrim("title")) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_thread_chapters_turn" ON "assistant_thread_chapters" ("organization_id", "business_unit_id", "thread_id", "turn_id");

CREATE INDEX IF NOT EXISTS "idx_assistant_thread_chapters_turn" ON "assistant_thread_chapters" ("turn_id", "organization_id", "business_unit_id");

COMMENT ON TABLE "assistant_thread_chapters" IS 'The chapters of a Desk conversation: each starts at the turn that opened it and runs to the next, named by the chapter model, the question that opened it, the schedule that ran it or the person';

COMMENT ON COLUMN "assistant_thread_chapters"."turn_id" IS 'The turn that opens the chapter; its first saved message is where the chapter starts';

COMMENT ON COLUMN "assistant_thread_chapters"."title_source" IS 'Who named the chapter: the chapter model, the opening question, a scheduled run or the person, whose name is never replaced';

ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "turn_id" VARCHAR(100);

COMMENT ON COLUMN "assistant_messages"."turn_id" IS 'The assistant turn that saved the message; empty on cards written outside a turn';

ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "auto_chapters_off" BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN "agent_controls"."auto_chapters_off" IS 'Desk conversations are no longer split into chapters and named by a model; people can still start chapters by hand';

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'FactoringRemittanceExtract', 'AgentReflection', 'AgentDrafting', 'AgentProfile', 'AgentRecommendation', 'ConversationChapters'));
