-- What a person reads and changes on an agent's capabilities page: who set the
-- agent up, the most records one change may touch, the hours it may change
-- things on its own, the topic each hand-off covers, and the tools switched
-- off there so they can be offered back.
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "created_by_id" VARCHAR(100),
    ADD COLUMN IF NOT EXISTS "max_change_items" INTEGER NOT NULL DEFAULT 500,
    ADD COLUMN IF NOT EXISTS "business_hours_only" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "business_hours_start" INTEGER NOT NULL DEFAULT 420,
    ADD COLUMN IF NOT EXISTS "business_hours_end" INTEGER NOT NULL DEFAULT 1080,
    ADD COLUMN IF NOT EXISTS "business_hours_timezone" VARCHAR(100),
    ADD COLUMN IF NOT EXISTS "delegate_topics" JSONB,
    ADD COLUMN IF NOT EXISTS "disabled_tool_names" TEXT[];

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_max_change_items";

--bun:split

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_max_change_items" CHECK ("max_change_items" BETWEEN 1 AND 10000);

--bun:split

ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_business_hours";

--bun:split

ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "ck_agent_definitions_business_hours" CHECK (
        "business_hours_start" >= 0
        AND "business_hours_end" <= 1440
        AND "business_hours_start" < "business_hours_end"
    );

COMMENT ON COLUMN "agent_definitions"."created_by_id" IS 'The person who set the agent up; null for platform agents and agents made before it was kept';
COMMENT ON COLUMN "agent_definitions"."max_change_items" IS 'The most records one change by the agent may touch; a bigger batch is refused with an instruction to split it';
COMMENT ON COLUMN "agent_definitions"."business_hours_only" IS 'Outside the business-hours window the agent''s changes are held as proposals instead of running on their own';
COMMENT ON COLUMN "agent_definitions"."business_hours_start" IS 'Start of the business-hours window, in minutes after midnight';
COMMENT ON COLUMN "agent_definitions"."business_hours_end" IS 'End of the business-hours window, in minutes after midnight, exclusive';
COMMENT ON COLUMN "agent_definitions"."business_hours_timezone" IS 'Zone the business-hours window is read in; null uses the organization''s';
COMMENT ON COLUMN "agent_definitions"."delegate_topics" IS 'Per delegate agent id, the label naming which questions go to it';
COMMENT ON COLUMN "agent_definitions"."disabled_tool_names" IS 'Tools switched off on the capabilities page, kept so they can be offered back; never part of the grant';
