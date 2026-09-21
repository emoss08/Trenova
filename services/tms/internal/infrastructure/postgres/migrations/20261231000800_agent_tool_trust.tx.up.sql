-- One row per agent and tool: how the people deciding on that agent's
-- proposals for that tool have been deciding. The streak is the count of
-- consecutive clean approvals; a change, a rejection or a failed execution
-- sends it back to zero. When the organization has earned autonomy switched on
-- and the streak reaches its threshold, the tool moves up one tier on the
-- agent, and the row remembers that the tier was earned rather than chosen so
-- that a later rejection can take it back without touching a person's choice.
CREATE TABLE IF NOT EXISTS "agent_tool_trust" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "tool_name" varchar(100) NOT NULL,
    "streak" integer NOT NULL DEFAULT 0,
    "approvals" integer NOT NULL DEFAULT 0,
    "modifications" integer NOT NULL DEFAULT 0,
    "rejections" integer NOT NULL DEFAULT 0,
    "execution_failures" integer NOT NULL DEFAULT 0,
    "earned_tier" agent_autonomy_tier_enum,
    "last_decision_at" bigint,
    "promoted_at" bigint,
    "demoted_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_agent_tool_trust" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_tool_trust_counts" CHECK (
        "streak" >= 0 AND "approvals" >= 0 AND "modifications" >= 0
        AND "rejections" >= 0 AND "execution_failures" >= 0
    ),
    CONSTRAINT "fk_agent_tool_trust_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_tool_trust_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_tool_trust_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Every decision lands on exactly one row per agent and tool, so the ledger
-- is written as an upsert against this key.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_tool_trust_agent_tool"
    ON "agent_tool_trust"("organization_id", "business_unit_id", "agent_definition_id", "tool_name");

--bun:split
-- The organization decides whether trust is allowed to change a tier at all,
-- and how many clean approvals in a row it takes. Off by default: an agent
-- gaining reach on its own is something an organization opts into.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "earned_autonomy" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "promotion_threshold" integer NOT NULL DEFAULT 10;

--bun:split
ALTER TABLE "agent_controls"
    ADD CONSTRAINT "chk_agent_controls_promotion_threshold" CHECK ("promotion_threshold" BETWEEN 1 AND 1000);

COMMENT ON TABLE "agent_tool_trust" IS 'Per agent and tool record of how proposals were decided; drives earned autonomy';

COMMENT ON COLUMN "agent_tool_trust"."streak" IS 'Consecutive approvals without a change, rejection or failed execution';

COMMENT ON COLUMN "agent_tool_trust"."earned_tier" IS 'Tier the ledger granted this tool on the agent, when the current tier was earned rather than chosen';

COMMENT ON COLUMN "agent_controls"."earned_autonomy" IS 'Whether a clean streak of approvals may move a tool up one tier on an agent';

COMMENT ON COLUMN "agent_controls"."promotion_threshold" IS 'Consecutive clean approvals needed before a tool moves up a tier';
