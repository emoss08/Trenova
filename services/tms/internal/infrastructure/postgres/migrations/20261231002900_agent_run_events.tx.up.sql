-- An agent's account of what it did, kept.
--
-- The runtime has always emitted this: which tool it reached for, what came
-- back, what it refused, when it gave up. Nothing was writing it down. A
-- conversation's events went to a redis stream trimmed a quarter of an hour
-- after the reply ended, and a background run's went nowhere at all — they
-- arrived at a function that used them to say "still working" and dropped them.
-- What survived a background run was its final reply cut to two thousand
-- characters.
--
-- These rows are append-only and are never revised. The step ledger beside them
-- answers "has this already run", which is why its rows are claimed and then
-- settled; an event answers "what happened", which nothing later can change.
-- The two join on step_key wherever a tool is involved.
CREATE TABLE IF NOT EXISTS "agent_run_events"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "owner_kind" VARCHAR(20) NOT NULL,
    "owner_id" VARCHAR(100) NOT NULL,
    "sequence" INTEGER NOT NULL,
    "kind" VARCHAR(40) NOT NULL,
    "step_key" VARCHAR(120) NOT NULL DEFAULT '',
    "call_id" VARCHAR(200) NOT NULL DEFAULT '',
    "payload" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "truncated" BOOLEAN NOT NULL DEFAULT FALSE,
    "occurred_at" BIGINT NOT NULL,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_agent_run_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_agent_run_events_owner_kind" CHECK ("owner_kind" IN ('AgentRun', 'AssistantTurn')),
    CONSTRAINT "ck_agent_run_events_sequence" CHECK ("sequence" >= 1),
    CONSTRAINT "fk_agent_run_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_run_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Reading one run's trajectory in order, which is the only way it is ever read.
-- Unique because two events sharing a sequence would make that order a lie, and
-- a writer that lost count should fail rather than quietly interleave.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_run_events_sequence" ON "agent_run_events"("organization_id", "owner_id", "sequence");

--bun:split
-- Deliberately not scoped to a tenant: the retention sweep walks by age across
-- every organization, and a leading tenant column would make it seek per tenant
-- rather than scan once.
CREATE INDEX IF NOT EXISTS "idx_agent_run_events_pruning" ON "agent_run_events"("created_at");
