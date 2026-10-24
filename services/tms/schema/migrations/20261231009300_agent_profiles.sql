-- Agents knew an organization by its name and a person by their name and roles.
-- A profile is what they are told besides: worked out each night from the
-- organization's own records (what it runs, for whom, on which lanes, with what
-- fleet and which integrations) or from what a person works on, and written up
-- by a model from those facts and the guidance people gave it. Every version is
-- kept, so what an agent was told on a given day can be read back.
CREATE TABLE IF NOT EXISTS "agent_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "subject" varchar(20) NOT NULL,
    "user_id" varchar(100),
    "status" varchar(20) NOT NULL,
    "trigger" varchar(20) NOT NULL,
    "requested_by_id" varchar(100),
    "content" text,
    "facts" jsonb NOT NULL DEFAULT '[]',
    "guidance_ids" jsonb NOT NULL DEFAULT '[]',
    "digest" varchar(64) NOT NULL,
    "model" varchar(200),
    "provider_id" varchar(100),
    "input_tokens" integer NOT NULL DEFAULT 0,
    "output_tokens" integer NOT NULL DEFAULT 0,
    "error_message" text,
    "checked_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_profiles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_profiles_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_profiles_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_profiles_subject" CHECK ("subject" IN ('Organization', 'User')),
    CONSTRAINT "ck_agent_profiles_subject_user" CHECK (("subject" = 'User') = ("user_id" IS NOT NULL)),
    CONSTRAINT "ck_agent_profiles_status" CHECK ("status" IN ('Current', 'Superseded', 'Failed')),
    CONSTRAINT "ck_agent_profiles_trigger" CHECK ("trigger" IN ('Scheduled', 'Requested', 'Guidance')),
    CONSTRAINT "ck_agent_profiles_content" CHECK ("status" = 'Failed' OR length(btrim(coalesce("content", ''))) > 0),
    CONSTRAINT "ck_agent_profiles_failure" CHECK (("status" = 'Failed') = ("error_message" IS NOT NULL)),
    CONSTRAINT "ck_agent_profiles_tokens" CHECK ("input_tokens" >= 0 AND "output_tokens" >= 0)
);

-- One version is what agents are told, per organization and per person.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_profiles_current_organization" ON "agent_profiles"("organization_id", "business_unit_id")
WHERE
    "status" = 'Current' AND "subject" = 'Organization';

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_profiles_current_user" ON "agent_profiles"("organization_id", "business_unit_id", "user_id")
WHERE
    "status" = 'Current' AND "subject" = 'User';

CREATE INDEX IF NOT EXISTS "idx_agent_profiles_history" ON "agent_profiles"("organization_id", "business_unit_id", "subject", "user_id", "created_at" DESC, "id" DESC);

COMMENT ON TABLE "agent_profiles" IS 'What agents are told about an organization or a person: a model''s write-up of facts worked out from the records and of guidance people gave, one row per version';

COMMENT ON COLUMN "agent_profiles"."facts" IS 'The facts the write-up was made from, each worked out by a query over the organization''s records rather than by a model';

COMMENT ON COLUMN "agent_profiles"."guidance_ids" IS 'The guidance in force when the version was written';

COMMENT ON COLUMN "agent_profiles"."digest" IS 'A hash of the facts, the guidance and the prompt version; a nightly pass that finds the same digest writes nothing new';

COMMENT ON COLUMN "agent_profiles"."checked_at" IS 'When the facts were last worked out and found to give this version';

-- Guidance is what people say about a profile: what the facts cannot show, or
-- how to read them ("the Laredo lane is seasonal", "I only handle the Midwest
-- accounts"). The model never edits the profile on a person's behalf; it writes
-- the guidance from what they said, and the next version honours it.
CREATE TABLE IF NOT EXISTS "agent_profile_guidance"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "subject" varchar(20) NOT NULL,
    "user_id" varchar(100),
    "content" text NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "source" varchar(20) NOT NULL,
    "created_by_id" varchar(100) NOT NULL,
    "thread_id" varchar(100),
    "retired_by_id" varchar(100),
    "retired_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_profile_guidance" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_profile_guidance_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_profile_guidance_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_profile_guidance_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_profile_guidance_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_profile_guidance_retired_by" FOREIGN KEY ("retired_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_profile_guidance_subject" CHECK ("subject" IN ('Organization', 'User')),
    CONSTRAINT "ck_agent_profile_guidance_subject_user" CHECK (("subject" = 'User') = ("user_id" IS NOT NULL)),
    CONSTRAINT "ck_agent_profile_guidance_status" CHECK ("status" IN ('Active', 'Retired')),
    CONSTRAINT "ck_agent_profile_guidance_retired" CHECK (("status" = 'Retired') = ("retired_at" IS NOT NULL)),
    CONSTRAINT "ck_agent_profile_guidance_source" CHECK ("source" IN ('Person', 'Agent')),
    CONSTRAINT "ck_agent_profile_guidance_content" CHECK (length(btrim("content")) BETWEEN 1 AND 500)
);

CREATE INDEX IF NOT EXISTS "idx_agent_profile_guidance_active" ON "agent_profile_guidance"("organization_id", "business_unit_id", "subject", "user_id", "created_at")
WHERE
    "status" = 'Active';

CREATE INDEX IF NOT EXISTS "idx_agent_profile_guidance_history" ON "agent_profile_guidance"("organization_id", "business_unit_id", "subject", "user_id", "created_at" DESC, "id" DESC);

COMMENT ON TABLE "agent_profile_guidance" IS 'What people told the agents about how to read an organization''s or their own profile; the next version of the profile honours every active line';

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_feature" CHECK ("feature" IN ('AgentTurn', 'AgentEvaluation', 'TableQuery', 'FormulaGenerate', 'FormulaExplain', 'ShipmentImportChat', 'DocumentIntelligenceRoute', 'DocumentIntelligenceExtract', 'AccountingMapping', 'AgentReflection', 'AgentDrafting', 'AgentProfile'));

SELECT
    trenova_rls.reconcile();
