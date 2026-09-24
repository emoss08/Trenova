ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "data_access_ceiling" VARCHAR(20) NOT NULL DEFAULT 'Internal';

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_data_access_ceiling";

--bun:split

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_data_access_ceiling" CHECK ("data_access_ceiling" IN ('Internal', 'Restricted'));

COMMENT ON COLUMN "agent_definitions"."data_access_ceiling" IS 'The most sensitive fields the agent''s tools may read: Internal withholds amounts, pay and other Restricted fields, Restricted shows them. A run a person is in never reads past that person''s own access';

--bun:split

UPDATE "agent_definitions"
SET "data_access_ceiling" = 'Restricted'
WHERE "trigger_mode" = 'Chat' OR "template" = 'CashApplication';
