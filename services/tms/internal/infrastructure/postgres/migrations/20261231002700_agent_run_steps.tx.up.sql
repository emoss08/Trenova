-- A run is one activity the length of a whole conversation with a model, and
-- an activity that fails is retried from its start. Without a record of what
-- the first attempt did, the retry asks the model again and runs every write
-- again -- including the ones that had already succeeded. A tool's declared
-- idempotency key does not prevent this: it is checked for presence and, with
-- two exceptions that forward it to an email provider, never looked up.
--
-- These rows are that record. A step is claimed before its tool runs and
-- settled after, so a row left 'Started' names a write whose outcome nobody
-- recorded -- the one case a retry must report rather than repeat.
CREATE TABLE IF NOT EXISTS "agent_run_steps"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- owner_kind names which ledger the step belongs to: a background run's
    -- agent_runs row, or a conversation turn's. One table because the rule is
    -- the same for both, and no foreign key because one column cannot point at
    -- two tables -- retention is swept rather than cascaded.
    "owner_kind" varchar(20) NOT NULL,
    "owner_id" varchar(100) NOT NULL,
    "attempt" integer NOT NULL DEFAULT 1,
    "kind" varchar(20) NOT NULL,
    "status" varchar(20) NOT NULL,
    -- step_key identifies the operation across attempts: the owner, the tool
    -- and the canonical arguments. Deliberately not the provider's call id,
    -- which is minted fresh on every completion and so names nothing stable.
    "step_key" varchar(120) NOT NULL,
    "tool_name" varchar(200) NOT NULL DEFAULT '',
    "call_id" varchar(200) NOT NULL DEFAULT '',
    "arguments" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "outcome" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_run_steps" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_run_steps_owner_kind" CHECK ("owner_kind" IN ('AgentRun', 'AssistantTurn')),
    CONSTRAINT "ck_agent_run_steps_kind" CHECK ("kind" IN ('Tool', 'Completion')),
    CONSTRAINT "ck_agent_run_steps_status" CHECK ("status" IN ('Started', 'Completed', 'Failed')),
    CONSTRAINT "fk_agent_run_steps_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_run_steps_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- The claim itself. A second attempt inserting the same key collides, and the
-- collision is the proof that the work was already begun. Only tool steps are
-- claimed; a model reply is filed under a key unique to its own row, so this
-- never collapses two completions.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_run_steps_key" ON "agent_run_steps"("organization_id", "owner_id", "step_key");

--bun:split
-- Reading a run's ledger back, in the order it happened.
CREATE INDEX IF NOT EXISTS "idx_agent_run_steps_owner" ON "agent_run_steps"("organization_id", "business_unit_id", "owner_id", "created_at");

--bun:split
-- Steps are an execution artifact, not a record of account, and are swept.
CREATE INDEX IF NOT EXISTS "idx_agent_run_steps_pruning" ON "agent_run_steps"("created_at");
