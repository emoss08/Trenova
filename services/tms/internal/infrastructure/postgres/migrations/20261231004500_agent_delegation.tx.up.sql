ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "delegate_ids" TEXT[];

COMMENT ON COLUMN "agent_definitions"."delegate_ids" IS 'Agent definitions in the same organization and business unit this agent may hand a task to in a conversation; removed from every allowlist when the named agent is deleted';

--bun:split

ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "agent_definition_id" VARCHAR(100);

ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "delegate_call_id" VARCHAR(200);

COMMENT ON COLUMN "assistant_messages"."agent_definition_id" IS 'The agent that took this step, set only on the steps of an agent the conversation''s agent handed a task to';

COMMENT ON COLUMN "assistant_messages"."delegate_call_id" IS 'The delegate_task call this step answers, set only on the steps of an agent the conversation''s agent handed a task to';

--bun:split

ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated'));

COMMENT ON COLUMN "assistant_messages"."kind" IS 'Message for what a person or the model wrote; DecisionNote for the input of the turn that follows a decision on a proposal; Delegated for a step another agent took on a task the conversation''s agent handed it, which is never replayed to the model';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_messages_delegate_call"
    ON "assistant_messages" ("organization_id", "business_unit_id", "thread_id", "delegate_call_id")
    WHERE "delegate_call_id" IS NOT NULL;
