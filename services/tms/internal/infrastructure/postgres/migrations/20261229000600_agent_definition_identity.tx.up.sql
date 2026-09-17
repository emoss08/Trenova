-- How an agent is drawn wherever it appears. An organization picks these, so an
-- agent it wrote is as recognizable in a list as one the platform ships.
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "icon" VARCHAR(40),
    ADD COLUMN IF NOT EXISTS "accent" VARCHAR(20);

--bun:split
ALTER TABLE "agent_definitions"
    ADD CONSTRAINT "chk_agent_definitions_accent" CHECK (
        "accent" IS NULL
        OR "accent" IN ('indigo', 'teal', 'amber', 'rose', 'emerald', 'sky', 'violet', 'slate')
    );

--bun:split
COMMENT ON COLUMN "agent_definitions"."icon" IS 'Chosen icon name; null falls back to the starter template''s icon';

--bun:split
COMMENT ON COLUMN "agent_definitions"."accent" IS 'Chosen accent name; null falls back to an accent derived from the agent id';
