-- How many questions one person may ask the agents in a month. Nought leaves
-- it unlimited, which is how every organization starts.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "person_monthly_messages" INTEGER NOT NULL DEFAULT 0;

--bun:split
COMMENT ON COLUMN "agent_controls"."person_monthly_messages" IS 'Questions one person may ask the agents each calendar month (UTC); 0 is unlimited';
