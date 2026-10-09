--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "agent_waits"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "kind" varchar(50) NOT NULL,
    "condition" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "watch_id" varchar(100),
    "description" text NOT NULL,
    "then_note" text,
    "agent_definition_id" varchar(100) NOT NULL,
    "thread_id" varchar(100),
    "user_id" varchar(100),
    "run_id" varchar(100),
    "subject_type" varchar(50),
    "subject_id" varchar(100),
    "taint" jsonb,
    "status" varchar(50) NOT NULL DEFAULT 'Waiting',
    "due_at" bigint,
    "expires_at" bigint NOT NULL,
    "resolved_at" bigint,
    "outcome" text,
    "resumed_turn_id" varchar(100),
    "resumed_run_id" varchar(100),
    "workflow_id" varchar(200),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_agent_waits" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_waits_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_waits_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_waits_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow')),
    CONSTRAINT "ck_agent_waits_status" CHECK ("status" IN ('Waiting', 'Met', 'TimedOut', 'Cancelled', 'Failed')),
    CONSTRAINT "ck_agent_waits_condition" CHECK (jsonb_typeof("condition") = 'object'),
    CONSTRAINT "ck_agent_waits_owner" CHECK ("thread_id" IS NOT NULL OR ("run_id" IS NOT NULL AND "subject_id" IS NOT NULL))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_waits_open_watch" ON "agent_waits"("organization_id", "business_unit_id", "kind", "watch_id") WHERE "status" = 'Waiting';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_waits_thread" ON "agent_waits"("organization_id", "business_unit_id", "thread_id", "created_at" DESC) WHERE "thread_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_waits_definition_open" ON "agent_waits"("organization_id", "business_unit_id", "agent_definition_id") WHERE "status" = 'Waiting';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_waits_business_unit" ON "agent_waits"("business_unit_id");

--bun:split
COMMENT ON TABLE "agent_waits" IS 'Work an agent parked until something happens: a truck reaching a stop, a reply, an appointment or the end of free time coming round, a driver''s drive time running low, or a time. A Temporal workflow per wait holds the timer and the signal; when the wait ends it picks the work up as a turn of the conversation or a run of the same agent on the same record';

--bun:split
COMMENT ON COLUMN "agent_waits"."watch_id" IS 'The record whose change can end the wait: the move, the stop, the shipment, carrier or customer, the detention occurrence or the worker; null for a wait on a time';

--bun:split
ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split
ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule', 'Compaction', 'Handoff', 'HandoffBrief', 'Steer', 'WorldChange', 'WaitNote'));

--bun:split
ALTER TABLE "assistant_turns"
    DROP CONSTRAINT IF EXISTS "ck_assistant_turns_origin";

--bun:split
ALTER TABLE "assistant_turns"
    ADD CONSTRAINT "ck_assistant_turns_origin" CHECK ("origin" IN ('Person', 'DecisionFollowUp', 'Scheduled', 'Compaction', 'WaitResolved'));

--bun:split
ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_trigger";

--bun:split
ALTER TABLE "agent_runs"
    ADD CONSTRAINT "ck_agent_runs_trigger" CHECK ("trigger" IN ('Manual', 'Chat', 'Scheduled', 'Event', 'Continuous', 'Wait'));

--bun:split
SELECT trenova_rls.reconcile();
