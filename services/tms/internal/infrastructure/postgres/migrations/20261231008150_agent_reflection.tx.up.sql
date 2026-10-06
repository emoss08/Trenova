-- An agent now looks back over its own work once a conversation goes quiet or a
-- background run is settled, and keeps what it learned: a stated preference the
-- turn did not save, a fact it had to find out, or a procedure, the sequence of
-- steps that worked here. Each look back is recorded, whatever it kept, so what
-- an agent taught itself can be traced to the turns it read.
CREATE TABLE IF NOT EXISTS "agent_reflections"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "subject_type" varchar(20) NOT NULL,
    "thread_id" varchar(100),
    "run_id" varchar(100),
    "user_id" varchar(100),
    "from_sequence" integer NOT NULL DEFAULT 0,
    "through_sequence" integer NOT NULL DEFAULT 0,
    "status" varchar(20) NOT NULL DEFAULT 'Running',
    "skip_reason" varchar(30),
    "signals" jsonb NOT NULL DEFAULT '[]',
    "changes" jsonb NOT NULL DEFAULT '[]',
    "notes" text,
    "tainted" boolean NOT NULL DEFAULT FALSE,
    "model" varchar(200),
    "provider_id" varchar(100),
    "input_tokens" integer NOT NULL DEFAULT 0,
    "output_tokens" integer NOT NULL DEFAULT 0,
    "error_message" text,
    "finished_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_reflections" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_reflections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_reflections_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_reflections_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_reflections_subject_type" CHECK ("subject_type" IN ('Thread', 'Run')),
    CONSTRAINT "ck_agent_reflections_subject" CHECK (("subject_type" <> 'Thread' OR "thread_id" IS NOT NULL) AND ("subject_type" <> 'Run' OR "run_id" IS NOT NULL)),
    CONSTRAINT "ck_agent_reflections_status" CHECK ("status" IN ('Running', 'Skipped', 'Completed', 'Failed')),
    CONSTRAINT "ck_agent_reflections_skip_reason" CHECK ("skip_reason" IS NULL OR "skip_reason" IN ('NoSignal', 'NothingToRead', 'LearningOff', 'AgentUnavailable', 'OverBudget')),
    CONSTRAINT "ck_agent_reflections_skip_pair" CHECK (("status" = 'Skipped') = ("skip_reason" IS NOT NULL)),
    CONSTRAINT "ck_agent_reflections_window" CHECK ("from_sequence" >= 0 AND "through_sequence" >= "from_sequence"),
    CONSTRAINT "ck_agent_reflections_tokens" CHECK ("input_tokens" >= 0 AND "output_tokens" >= 0)
);

--bun:split
-- One look back per stretch of a conversation and one per background run: the
-- workflow that claims it retries into the same row rather than a second one.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_reflections_thread_window" ON "agent_reflections"("organization_id", "business_unit_id", "thread_id", "through_sequence")
WHERE
    "subject_type" = 'Thread';

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_reflections_run" ON "agent_reflections"("organization_id", "business_unit_id", "run_id")
WHERE
    "subject_type" = 'Run';

--bun:split
-- The next look back at a conversation starts after the last one that did not
-- fail, and AI Control lists an agent's newest first.
CREATE INDEX IF NOT EXISTS "idx_agent_reflections_thread_latest" ON "agent_reflections"("organization_id", "business_unit_id", "thread_id", "through_sequence" DESC)
WHERE
    "subject_type" = 'Thread' AND "status" <> 'Failed';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_reflections_agent_created" ON "agent_reflections"("organization_id", "business_unit_id", "agent_definition_id", "created_at" DESC, "id" DESC);

--bun:split
COMMENT ON TABLE "agent_reflections" IS 'One time an agent looked back over a stretch of a conversation or a settled background run and decided what, if anything, to keep as memory';

--bun:split
COMMENT ON COLUMN "agent_reflections"."signals" IS 'What in the work made it worth a look: a tool that failed and then worked, a person correcting the agent or stating how they want something done, a proposal changed or refused, a thumbs-down, a long task';

--bun:split
COMMENT ON COLUMN "agent_reflections"."changes" IS 'Each memory the look back kept, offered, refreshed or was refused, with why it was refused';

--bun:split
COMMENT ON COLUMN "agent_reflections"."from_sequence" IS 'For a conversation, the first message sequence read; the look back reads every message after the previous one that did not fail';

--bun:split
-- Procedure is how a task has been done here: the steps that worked, learned by
-- doing it or dictated by a person.
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_kind";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_kind" CHECK ("kind" IN ('Instruction', 'Fact', 'Correction', 'Procedure'));

--bun:split
COMMENT ON COLUMN "agent_memories"."kind" IS 'Instruction to follow, Fact to weigh, Correction learned from a decision on a proposal, or Procedure: the steps that worked for a task here';

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_source";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision', 'Feedback', 'Reflection'));

--bun:split
COMMENT ON COLUMN "agent_memories"."source" IS 'Who recorded it: a person, an agent through its remember tool, a decision on a proposal, ratings people gave an agent''s output, or an agent looking back over its own work';

--bun:split
ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "reflection_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "supersedes_id" varchar(100);

--bun:split
COMMENT ON COLUMN "agent_memories"."reflection_id" IS 'The look back that kept or offered the memory, for one whose source is Reflection';

--bun:split
COMMENT ON COLUMN "agent_memories"."supersedes_id" IS 'The memory this one replaces: once this one is active the other is retired, in the same statement that made it active';

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_reflection";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_reflection" CHECK ("source" <> 'Reflection' OR "reflection_id" IS NOT NULL);

--bun:split
ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_supersedes_self";

--bun:split
ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_supersedes_self" CHECK ("supersedes_id" IS NULL OR "supersedes_id" <> "id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_memories_reflection" ON "agent_memories"("organization_id", "business_unit_id", "reflection_id")
WHERE
    "reflection_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_memories_supersedes" ON "agent_memories"("organization_id", "business_unit_id", "supersedes_id")
WHERE
    "supersedes_id" IS NOT NULL;

--bun:split
-- Learning is on unless an organization or an agent turns it off: what it keeps
-- still follows each person's saving preference and is held for a person when
-- the work read outside content.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "learning_off" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN "agent_controls"."learning_off" IS 'Agents no longer look back over their work to keep what they learned as memory';

--bun:split
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "learning_off" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN "agent_definitions"."learning_off" IS 'This agent no longer looks back over its work to keep what it learned, whatever its organization chose';

--bun:split
ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

--bun:split
ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'AgentReflection'));

--bun:split
SELECT
    trenova_rls.reconcile();
