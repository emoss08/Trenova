-- What an organization has told its agents, kept between runs. A standing
-- instruction ("this customer needs the POD within a day"), a fact an agent
-- was told in conversation, or a correction learned from a person changing or
-- rejecting a proposal. Each is read back into the prompt of every agent that
-- asks for memory, and can be scoped to one customer, location, driver or
-- carrier, or to one tool.
CREATE TABLE IF NOT EXISTS "agent_memories" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "kind" varchar(20) NOT NULL,
    "source" varchar(20) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "subject_type" varchar(30),
    "subject_id" varchar(100),
    "subject_label" varchar(200),
    "tool_name" varchar(100),
    "content" text NOT NULL,
    "agent_definition_id" varchar(100),
    "source_run_id" varchar(100),
    "source_proposal_id" varchar(100),
    "created_by_user_id" varchar(100),
    "retired_by_user_id" varchar(100),
    "retired_at" bigint,
    "expires_at" bigint,
    "use_count" integer NOT NULL DEFAULT 0,
    "last_used_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_agent_memories" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_memories_kind" CHECK ("kind" IN ('Instruction', 'Fact', 'Correction')),
    CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision')),
    CONSTRAINT "chk_agent_memories_status" CHECK ("status" IN ('Active', 'Retired')),
    CONSTRAINT "chk_agent_memories_subject_type" CHECK ("subject_type" IS NULL OR "subject_type" IN ('Customer', 'Location', 'Worker', 'Carrier')),
    CONSTRAINT "chk_agent_memories_subject" CHECK (("subject_type" IS NULL) = ("subject_id" IS NULL)),
    CONSTRAINT "chk_agent_memories_use_count" CHECK ("use_count" >= 0),
    CONSTRAINT "fk_agent_memories_created_by" FOREIGN KEY ("created_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_memories_retired_by" FOREIGN KEY ("retired_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_memories_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_memories_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Every prompt build asks for the organization's active memories, by subject
-- and by tool; the activity list asks by status.
CREATE INDEX IF NOT EXISTS "idx_agent_memories_active"
    ON "agent_memories"("organization_id", "business_unit_id", "status", "subject_type", "subject_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_memories_tool"
    ON "agent_memories"("organization_id", "business_unit_id", "tool_name")
    WHERE "tool_name" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_memories_created"
    ON "agent_memories"("organization_id", "business_unit_id", "created_at" DESC);

COMMENT ON TABLE "agent_memories" IS 'Standing instructions, facts and corrections an organization keeps for its agents between runs';

COMMENT ON COLUMN "agent_memories"."kind" IS 'Instruction to follow, Fact to weigh, or Correction learned from a decision on a proposal';

COMMENT ON COLUMN "agent_memories"."source" IS 'Who recorded it: a person, an agent through its remember tool, or a decision on a proposal';

COMMENT ON COLUMN "agent_memories"."tool_name" IS 'Set when the memory is about one tool, such as a correction to how it was proposed';

COMMENT ON COLUMN "agent_memories"."use_count" IS 'How many prompts have carried this memory';
