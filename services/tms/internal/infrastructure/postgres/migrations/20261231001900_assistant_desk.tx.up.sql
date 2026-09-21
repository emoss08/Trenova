-- The Desk is where a person works with the assistant rather than a corner
-- panel they open over another page. A conversation gains where it began
-- (a quick question from the palette is not listed until it is kept), a pin,
-- and the record it is about when it was opened from one.
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "origin" varchar(20) NOT NULL DEFAULT 'Panel';

--bun:split
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "pinned" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "subject_type" varchar(50);

--bun:split
ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "subject_id" varchar(100);

--bun:split
ALTER TABLE "assistant_threads"
    ADD CONSTRAINT "ck_assistant_threads_origin" CHECK ("origin" IN ('Panel', 'Desk', 'Ask', 'Watchtower', 'Briefing'));

--bun:split
-- The rail reads pinned conversations first, then the rest by recency.
CREATE INDEX IF NOT EXISTS "idx_assistant_threads_user_pinned" ON "assistant_threads"("organization_id", "business_unit_id", "user_id", "pinned", "last_message_at" DESC);

--bun:split
-- An artifact is what a turn produced besides words: the rows a report
-- preview returned, a run to download, the email an agent wants to send,
-- the plan it wants to carry out, the record it looked up. The transcript
-- refers to them; the Desk renders them beside it. A draft or a plan is a
-- view over the proposal or plan that carries the decision, never a second
-- copy of it, which is what proposal_id is for.
CREATE TABLE IF NOT EXISTS "assistant_artifacts"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "thread_id" varchar(100) NOT NULL,
    "message_id" varchar(100),
    "run_id" varchar(100),
    "proposal_id" varchar(100),
    "plan_id" varchar(100),
    "kind" varchar(40) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Ready',
    "title" varchar(200) NOT NULL,
    "payload" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "source_tool_call_id" varchar(200) NOT NULL DEFAULT '',
    "pinned" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_assistant_artifacts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_artifacts_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_artifacts_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_artifacts_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_artifacts_kind" CHECK ("kind" IN ('report_preview', 'report_run', 'email_draft', 'plan', 'entity_card', 'table_view', 'rate_explanation', 'dashboard_ref', 'briefing', 'inbound_message')),
    CONSTRAINT "ck_assistant_artifacts_status" CHECK ("status" IN ('Pending', 'Ready', 'Failed', 'Sent'))
);

--bun:split
-- One artifact per tool call and kind: a turn replayed or retried must not
-- leave two tables for one preview.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_artifacts_source" ON "assistant_artifacts"("thread_id", "source_tool_call_id", "kind")
WHERE "source_tool_call_id" <> '';

--bun:split
-- A proposal or plan is shown once in the pane however many turns mention it.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_artifacts_proposal" ON "assistant_artifacts"("thread_id", "proposal_id")
WHERE "proposal_id" IS NOT NULL;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_artifacts_plan" ON "assistant_artifacts"("thread_id", "plan_id")
WHERE "plan_id" IS NOT NULL;

--bun:split
-- The pane lists a thread's artifacts newest first, pinned ones on top.
CREATE INDEX IF NOT EXISTS "idx_assistant_artifacts_thread" ON "assistant_artifacts"("organization_id", "business_unit_id", "thread_id", "pinned", "created_at" DESC);

COMMENT ON TABLE "assistant_artifacts" IS 'What an assistant turn produced besides words: report rows, runs, drafts, plans and record cards, rendered beside the conversation';

COMMENT ON COLUMN "assistant_artifacts"."proposal_id" IS 'The proposal an email draft is a view over; the decision lives on the proposal, never here';
