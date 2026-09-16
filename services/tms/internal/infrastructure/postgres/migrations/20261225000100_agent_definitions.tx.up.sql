-- An organization configures agents by composing Trenova-owned parts: which
-- template, which subset of that template's tools, how much autonomy, and a
-- bounded focus note.
--
-- There is deliberately no system_prompt column. An organization that could
-- supply one could write "you are a general coding assistant" and undo every
-- boundary the product depends on. The focus note is delivered to the model as
-- fenced, explicitly non-authoritative data instead.
CREATE TABLE IF NOT EXISTS "agent_definitions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "kind" varchar(50) NOT NULL,
    "focus" text,
    "tool_names" text[],
    "autonomy_ceiling" varchar(50) NOT NULL DEFAULT 'Propose',
    "enabled" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_definitions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_definitions_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_definitions_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_definitions_kind" CHECK ("kind" IN ('DispatchAssistant', 'BillingAssistant', 'ComplianceAssistant', 'CustomerAssistant', 'GeneralAssistant')),
    CONSTRAINT "ck_agent_definitions_autonomy" CHECK ("autonomy_ceiling" IN ('Propose', 'ActWithApproval', 'AutoExecute')),
    -- Bounded in the database as well as in validation: the note is sent on every
    -- turn, and an unbounded one is usually an attempt to write a system prompt
    -- in disguise.
    CONSTRAINT "ck_agent_definitions_focus_length" CHECK ("focus" IS NULL OR length("focus") <= 2000),
    CONSTRAINT "ck_agent_definitions_tool_count" CHECK ("tool_names" IS NULL OR cardinality("tool_names") <= 32)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_definitions_org_name" ON "agent_definitions"("organization_id", "business_unit_id", lower("name"));

CREATE INDEX IF NOT EXISTS "idx_agent_definitions_lookup" ON "agent_definitions"("organization_id", "business_unit_id", "enabled");

--bun:split
COMMENT ON TABLE "agent_definitions" IS 'Per-organization agent configurations composed from Trenova-owned templates';

COMMENT ON COLUMN "agent_definitions"."kind" IS 'Trenova-owned template carrying the system prompt and the outer bound on reachable resources';

COMMENT ON COLUMN "agent_definitions"."focus" IS 'Organization preference note, delivered to the model as fenced non-authoritative data, never as system instruction';

COMMENT ON COLUMN "agent_definitions"."tool_names" IS 'Subset of the template''s tools; validated against the template''s allowed resources on save';

COMMENT ON COLUMN "agent_definitions"."autonomy_ceiling" IS 'Caps each tool''s own tier; can only restrict, never raise';
