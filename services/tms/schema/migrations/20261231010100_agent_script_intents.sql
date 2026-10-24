-- A write a script prepared is filed as an intent and approved as part of one
-- batch; the script's ledger step names which batch.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "script_step_key" varchar(120);

ALTER TABLE "agent_plans"
    ADD COLUMN IF NOT EXISTS "script_step_key" varchar(120);

CREATE INDEX IF NOT EXISTS "idx_agent_plans_script_step" ON "agent_plans"("organization_id", "script_step_key")
WHERE
    "script_step_key" IS NOT NULL;
