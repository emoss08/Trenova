-- The reads Nova makes answering a palette question, one row per call: the palette tool the
-- model called, the Desk read it went to and what that read was asked, how it ended and why,
-- and how long it took. A question's reads were kept nowhere but its stream, so a refused or
-- empty answer could not be traced afterwards. The AI audit trail reads these rows as the
-- question's tool calls; they go with their question.
CREATE TABLE IF NOT EXISTS "assistant_palette_tool_calls" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "question_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100) NOT NULL,
    "call_id" VARCHAR(200) NOT NULL,
    "sequence" INTEGER NOT NULL,
    "model" VARCHAR(200),
    "tool" VARCHAR(100) NOT NULL,
    "arguments" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "read" VARCHAR(200),
    "params" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "outcome" VARCHAR(20) NOT NULL,
    "reason" TEXT,
    "summary" TEXT,
    "records" INTEGER NOT NULL DEFAULT 0,
    "started_at_ms" BIGINT NOT NULL,
    "duration_ms" BIGINT NOT NULL DEFAULT 0,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_assistant_palette_tool_calls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_assistant_palette_tool_calls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_palette_tool_calls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_palette_tool_calls_question" FOREIGN KEY ("question_id", "organization_id", "business_unit_id") REFERENCES "assistant_palette_questions" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_palette_tool_calls_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_palette_tool_calls_outcome" CHECK ("outcome" IN ('Ran', 'Refused', 'Failed')),
    CONSTRAINT "ck_assistant_palette_tool_calls_counts" CHECK ("sequence" > 0 AND "records" >= 0 AND "duration_ms" >= 0)
);

-- A question's reads, in the order they were made.
CREATE UNIQUE INDEX IF NOT EXISTS "idx_assistant_palette_tool_calls_question_sequence" ON "assistant_palette_tool_calls" ("organization_id", "business_unit_id", "question_id", "sequence");

-- The AI audit projector reads every tenant's calls forward by (created_at, id).
CREATE INDEX IF NOT EXISTS "idx_assistant_palette_tool_calls_created" ON "assistant_palette_tool_calls" ("created_at", "id");

COMMENT ON TABLE "assistant_palette_tool_calls" IS 'Each read Nova made answering a palette question: the tool, the read it went to, its arguments, how it ended and why. What the read returned is never kept';

-- A palette question owns its model calls and tool calls on the trail, as a run or a turn does.
ALTER TABLE "ai_usage_records" DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_owner_kind";
ALTER TABLE "ai_usage_records" ADD CONSTRAINT "ck_ai_usage_records_owner_kind" CHECK ("owner_kind" IS NULL OR "owner_kind" IN ('AgentRun', 'AssistantTurn', 'PaletteQuestion'));

ALTER TABLE "ai_audit_events" DROP CONSTRAINT IF EXISTS "ck_ai_audit_events_owner_kind";
ALTER TABLE "ai_audit_events" ADD CONSTRAINT "ck_ai_audit_events_owner_kind" CHECK ("owner_kind" IS NULL OR "owner_kind" IN ('AgentRun', 'AssistantTurn', 'PaletteQuestion'));

ALTER TABLE "ai_audit_projector_state" DROP CONSTRAINT IF EXISTS "ck_ai_audit_projector_state_source";
ALTER TABLE "ai_audit_projector_state" ADD CONSTRAINT "ck_ai_audit_projector_state_source" CHECK ("source" IN ('agent_runs', 'assistant_turns', 'ai_usage_records', 'agent_run_steps', 'agent_run_events', 'agent_proposals', 'agent_decisions', 'assistant_palette_tool_calls'));
