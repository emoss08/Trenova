-- A memory can now be kept for one person or for everyone holding a role, as
-- well as for the whole organization or one agent. The Desk calls these "Just
-- you" and the person's team: roles are how people are grouped and
-- permissioned, so a team is a role. Each scope names its readers in its own
-- column, and the check keeps a scope from being saved without them.
ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "owner_user_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "role_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "source_thread_id" varchar(100);

COMMENT ON COLUMN "agent_memories"."owner_user_id" IS 'The person a User-scoped memory is kept for; only their conversations read it';

COMMENT ON COLUMN "agent_memories"."role_id" IS 'The role a Role-scoped memory is kept for; only conversations of people holding the role read it';

COMMENT ON COLUMN "agent_memories"."source_thread_id" IS 'The conversation the run that recorded the memory was answering, so the memory can say where it came from';

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_scope";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_scope" CHECK (
        "scope" IN ('Organization', 'Agent', 'User', 'Role')
        AND ("scope" <> 'Agent' OR "agent_definition_id" IS NOT NULL)
        AND ("scope" <> 'User' OR "owner_user_id" IS NOT NULL)
        AND ("scope" <> 'Role' OR "role_id" IS NOT NULL)
    );

COMMENT ON COLUMN "agent_memories"."scope" IS 'Who reads the memory: Organization for every agent, Agent for agent_definition_id alone, User for owner_user_id''s conversations alone, Role for the conversations of everyone holding role_id';

--bun:split
-- Paused is a memory a person set aside without forgetting it. Like Retired it
-- is never read by an agent; unlike Retired it stays in the person's list.
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_status";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_status" CHECK ("status" IN ('Active', 'Paused', 'Retired', 'Suggested', 'Dismissed'));

COMMENT ON COLUMN "agent_memories"."status" IS 'Active memories are read into prompts; Paused ones are set aside by a person and Retired ones forgotten, neither read; Suggested ones wait for a person to accept them, drawn from feedback or offered by an agent to someone who asked to be asked first; Dismissed ones were refused. Only Active is ever read by an agent';

--bun:split
-- Memories an agent recorded before this knew their run, and an assistant run
-- names its conversation as its subject.
UPDATE "agent_memories" AS m
SET "source_thread_id" = r."subject_id"
FROM "agent_runs" AS r
WHERE m."source_run_id" = r."id"
  AND m."source_thread_id" IS NULL
  AND r."subject_type" = 'AssistantThread';

--bun:split
-- A prompt reads one person's memories and their roles' alongside the
-- organization's, and the Desk lists them by scope.
CREATE INDEX IF NOT EXISTS "idx_agent_memories_owner"
    ON "agent_memories" ("organization_id", "business_unit_id", "owner_user_id", "created_at" DESC)
    WHERE "scope" = 'User';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_role"
    ON "agent_memories" ("organization_id", "business_unit_id", "role_id", "created_at" DESC)
    WHERE "scope" = 'Role';

--bun:split

CREATE TABLE IF NOT EXISTS "agent_memory_preferences" (
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "saving_mode" varchar(20) NOT NULL DEFAULT 'Automatic',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_agent_memory_preferences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_agent_memory_preferences_user" UNIQUE ("organization_id", "business_unit_id", "user_id"),
    CONSTRAINT "chk_agent_memory_preferences_saving_mode" CHECK ("saving_mode" IN ('Automatic', 'AskFirst')),
    CONSTRAINT "fk_agent_memory_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_memory_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_memory_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

COMMENT ON TABLE "agent_memory_preferences" IS 'How each person wants memories an agent picks up in their conversations kept: saved and announced, or offered to them first';

COMMENT ON COLUMN "agent_memory_preferences"."saving_mode" IS 'Automatic saves what an agent records and says so in the conversation; AskFirst keeps it as a suggestion until the person accepts it';

--bun:split
-- A turn's last reply keeps which memories the turn used and which it saved,
-- so the conversation can show them again when it is read back.
ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "used_memory_ids" jsonb,
    ADD COLUMN IF NOT EXISTS "saved_memories" jsonb;

COMMENT ON COLUMN "assistant_messages"."used_memory_ids" IS 'On a turn''s last reply: the agent_memories the turn used, those its prompt carried and those recall_memory read back';

COMMENT ON COLUMN "assistant_messages"."saved_memories" IS 'On a turn''s last reply: each memory the turn kept or offered to keep through remember, with its call and whether it waits for the person';
