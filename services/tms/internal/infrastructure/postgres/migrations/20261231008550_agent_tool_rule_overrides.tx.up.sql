-- An organization holding one tool lower than the rule declared beside it in
-- code: a lower most-freedom tier, or more of what it returns treated as
-- outside text. A rule here never loosens the declared one.
CREATE TABLE IF NOT EXISTS "agent_tool_rule_overrides"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "tool_name" varchar(100) NOT NULL,
    "max_tier" varchar(30),
    "reads_external" varchar(20),
    "reason" text NOT NULL DEFAULT '',
    "updated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_tool_rule_overrides" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_tool_rule_overrides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_tool_rule_overrides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_tool_rule_overrides_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_tool_rule_overrides_max_tier" CHECK ("max_tier" IS NULL OR "max_tier" IN ('Propose', 'ActWithApproval', 'AutoExecute')),
    CONSTRAINT "ck_agent_tool_rule_overrides_reads_external" CHECK ("reads_external" IS NULL OR "reads_external" IN ('never', 'marked', 'always'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_tool_rule_overrides_tool" ON "agent_tool_rule_overrides"("organization_id", "business_unit_id", "tool_name");

--bun:split
COMMENT ON TABLE "agent_tool_rule_overrides" IS 'An organization holding a tool lower than its declared rule';

--bun:split
SELECT trenova_rls.reconcile();
