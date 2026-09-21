-- An evaluation replays a recorded run against the agent as it is now, with
-- every write simulated, and compares what the replay would have done with
-- what the original proposed and how people decided on it. It is how a
-- change to an agent's instructions, tools or model is checked against
-- runs that already happened instead of against the next live one.
CREATE TABLE IF NOT EXISTS "agent_evaluations" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "source_run_id" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "trigger" varchar(20) NOT NULL,
    "subject_type" varchar(50) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "input" text,
    "definition_version" bigint NOT NULL DEFAULT 0,
    "prompt_version" varchar(100),
    "model" varchar(255),
    "provider_id" varchar(100),
    "reply" text,
    "actions" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "comparison" jsonb,
    "original_proposals" integer NOT NULL DEFAULT 0,
    "tool_calls_used" integer NOT NULL DEFAULT 0,
    "workflow_id" varchar(255),
    "error_message" text,
    "requested_by_user_id" varchar(100),
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_agent_evaluations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_evaluations_status" CHECK ("status" IN ('Pending', 'Running', 'Completed', 'Failed')),
    CONSTRAINT "fk_agent_evaluations_run" FOREIGN KEY ("source_run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_evaluations_requested_by" FOREIGN KEY ("requested_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_evaluations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_evaluations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_evaluations_run"
    ON "agent_evaluations"("organization_id", "business_unit_id", "source_run_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_evaluations_definition"
    ON "agent_evaluations"("organization_id", "business_unit_id", "agent_definition_id", "created_at" DESC);

COMMENT ON TABLE "agent_evaluations" IS 'A recorded agent run replayed against the agent as it is now, writes simulated, compared with the original and its decisions';

COMMENT ON COLUMN "agent_evaluations"."actions" IS 'What the replay would have done: each tool call with its parameters and preview';

COMMENT ON COLUMN "agent_evaluations"."comparison" IS 'Each original proposal matched against the replay and judged by how a person decided on it';
