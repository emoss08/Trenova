-- A run that has read content written outside the organization is tainted:
-- an inbound email, an extracted document, an EDI file, a bank receipt, a
-- chat attachment, a memory another tainted run wrote. A tainted run never
-- makes a write that leaves the organization on its own. What it read is kept
-- on the run, on each proposal it raised, on the conversation it ran in and on
-- any memory it wrote, so the rule can be audited and a later turn inherits it.
ALTER TABLE "agent_runs"
    ADD COLUMN IF NOT EXISTS "tainted" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "taint" jsonb,
    ADD COLUMN IF NOT EXISTS "tainted_at" bigint;

COMMENT ON COLUMN "agent_runs"."tainted" IS 'The run read content written outside the organization; no write of it that leaves the organization ran without a person approving it';

COMMENT ON COLUMN "agent_runs"."taint" IS 'Where the outside content came from: up to sixteen marks, each a source (inbound_message, document, edi, bank_receipt, weather, attachment, memory, run_record), the tool call that read it and the record when known';

COMMENT ON COLUMN "agent_runs"."tainted_at" IS 'When the run was first recorded as tainted';

--bun:split

ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "tainted" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "taint" jsonb,
    ADD COLUMN IF NOT EXISTS "egress_class" varchar(30),
    ADD COLUMN IF NOT EXISTS "held_by" text[] NOT NULL DEFAULT '{}';

--bun:split

ALTER TABLE "agent_proposals"
    DROP CONSTRAINT IF EXISTS "chk_agent_proposals_egress_class";

--bun:split

ALTER TABLE "agent_proposals"
    ADD CONSTRAINT "chk_agent_proposals_egress_class" CHECK (
        "egress_class" IS NULL OR "egress_class" IN (
            'none', 'personal', 'internal', 'customer_visible', 'driver_visible',
            'external_recipient', 'money'
        )
    );

COMMENT ON COLUMN "agent_proposals"."tainted" IS 'The run had read outside content when this write was decided; if its egress class leaves the organization it runs only on a person''s approval';

COMMENT ON COLUMN "agent_proposals"."taint" IS 'The marks of the outside content the run had read, as on agent_runs.taint';

COMMENT ON COLUMN "agent_proposals"."egress_class" IS 'Where the write reaches: none, personal, internal, customer_visible, driver_visible, external_recipient or money; rewritten with the class it had as it ran. Null for a proposal recorded before it was kept';

COMMENT ON COLUMN "agent_proposals"."held_by" IS 'What held the write below running on its own: agent_ceiling, tool_max, egress_class, condition, tainted, tool_tier or personal_exemption';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_tainted"
    ON "agent_proposals" ("organization_id", "business_unit_id", "created_at" DESC)
    WHERE "tainted";

--bun:split

ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "taint" jsonb,
    ADD COLUMN IF NOT EXISTS "tainted_at" bigint;

COMMENT ON COLUMN "assistant_threads"."taint" IS 'The outside content the conversation has read, as on agent_runs.taint; every later turn in it, a decision''s follow-up included, opens with it';

COMMENT ON COLUMN "assistant_threads"."tainted_at" IS 'When a turn of the conversation first read outside content';

--bun:split

ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "scope" varchar(20) NOT NULL DEFAULT 'Organization',
    ADD COLUMN IF NOT EXISTS "tainted" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "taint_run_id" varchar(100);

--bun:split
-- An approved suggestion drawn from feedback is about the one agent whose
-- output was rated, so it is kept for that agent. Everything else, a memory an
-- agent wrote included, stays with the whole organization as it was read
-- before.
UPDATE "agent_memories"
SET "scope" = 'Agent'
WHERE "source" = 'Feedback'
  AND "agent_definition_id" IS NOT NULL;

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_scope";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_scope" CHECK (
        "scope" IN ('Organization', 'Agent')
        AND ("scope" = 'Organization' OR "agent_definition_id" IS NOT NULL)
    );

COMMENT ON COLUMN "agent_memories"."scope" IS 'Who reads the memory: Organization for every agent, Agent for agent_definition_id alone. A suggestion drawn from feedback is approved as Agent unless an administrator widens it';

COMMENT ON COLUMN "agent_memories"."tainted" IS 'Written by a run that had read outside content; a run that reads the memory back is tainted by it';

COMMENT ON COLUMN "agent_memories"."taint_run_id" IS 'The run whose outside content the memory carries, when it was written inside a run';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_agent_scope"
    ON "agent_memories" ("organization_id", "business_unit_id", "agent_definition_id")
    WHERE "scope" = 'Agent';

--bun:split
-- schedule_report and create_shipment may go no higher than ActWithApproval.
-- A tier stored above that, set on an agent or earned from its decisions,
-- would read as a promise the policy no longer keeps, so it is brought down.
UPDATE "agent_definitions"
SET "tool_tiers" = "tool_tiers" || jsonb_build_object('schedule_report', 'ActWithApproval')
WHERE "tool_tiers" ->> 'schedule_report' = 'AutoExecute';

--bun:split

UPDATE "agent_definitions"
SET "tool_tiers" = "tool_tiers" || jsonb_build_object('create_shipment', 'ActWithApproval')
WHERE "tool_tiers" ->> 'create_shipment' = 'AutoExecute';

--bun:split

UPDATE "agent_tool_trust"
SET "earned_tier" = 'ActWithApproval',
    "version" = "version" + 1,
    "updated_at" = extract(epoch from current_timestamp)::bigint
WHERE "tool_name" IN ('schedule_report', 'create_shipment')
  AND "earned_tier" = 'AutoExecute';
