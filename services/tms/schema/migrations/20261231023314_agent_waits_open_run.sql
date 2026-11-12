-- Whether a finished run left a wait open is asked once for every run that
-- ends having proposed nothing. One probe of the open waits by run, however
-- many waits the tenant has ever had.
CREATE INDEX IF NOT EXISTS "idx_agent_waits_open_run" ON "agent_waits"("organization_id", "business_unit_id", "run_id")
WHERE
    "status" = 'Waiting' AND "run_id" IS NOT NULL;
