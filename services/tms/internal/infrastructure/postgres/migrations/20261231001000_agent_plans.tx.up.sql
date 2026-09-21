-- A run that needs several writes used to produce several proposals, each
-- decided on its own. A plan is those proposals as one thing: approved once,
-- executed in the order the agent asked for them, stopped at the first step
-- that fails so the steps after it never run against a world the failed one
-- was supposed to change.
CREATE TABLE IF NOT EXISTS "agent_plans" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "title" varchar(200) NOT NULL,
    "summary" text,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "step_count" integer NOT NULL DEFAULT 0,
    "completed_steps" integer NOT NULL DEFAULT 0,
    "failed_step" integer,
    "failure_error" text,
    "decided_by_user_id" varchar(100),
    "decided_at" bigint,
    "expires_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_agent_plans" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_plans_status" CHECK ("status" IN ('Pending', 'Approved', 'Completed', 'Failed', 'Rejected', 'Expired')),
    CONSTRAINT "chk_agent_plans_steps" CHECK ("step_count" >= 0 AND "completed_steps" >= 0 AND "completed_steps" <= "step_count"),
    CONSTRAINT "fk_agent_plans_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "agent_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_plans_decided_by" FOREIGN KEY ("decided_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_plans_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_plans_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_plans_run" ON "agent_plans"("organization_id", "run_id");

--bun:split
-- The sweeper asks which pending plans have expired; the activity list asks
-- which are pending.
CREATE INDEX IF NOT EXISTS "idx_agent_plans_pending" ON "agent_plans"("organization_id", "status", "expires_at");

--bun:split
-- A proposal knows which plan it belongs to and where in the order it sits.
-- Null for a proposal that stands on its own.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "plan_id" varchar(100);

--bun:split
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "plan_step" integer NOT NULL DEFAULT 0;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_proposals_plan" ON "agent_proposals"("organization_id", "plan_id", "plan_step")
    WHERE "plan_id" IS NOT NULL;

COMMENT ON TABLE "agent_plans" IS 'Several proposals from one run, decided once and executed in order';

COMMENT ON COLUMN "agent_plans"."failed_step" IS 'The step whose execution failed and stopped the plan; null while none has';

COMMENT ON COLUMN "agent_proposals"."plan_id" IS 'The plan this proposal is a step of; null for a proposal decided on its own';

COMMENT ON COLUMN "agent_proposals"."plan_step" IS 'Position in the plan, from one; zero outside a plan';
