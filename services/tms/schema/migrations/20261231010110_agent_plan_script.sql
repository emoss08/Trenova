-- What the script that wrote a batch was asked to do and the code it ran,
-- kept with the plan once the run's steps are swept.
ALTER TABLE "agent_plans"
    ADD COLUMN IF NOT EXISTS "script" jsonb;
