-- Prompts kept with an agent for trying it again after an edit.
CREATE TABLE IF NOT EXISTS "agent_test_prompts"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "agent_definition_id" varchar(100) NOT NULL,
    "prompt" text NOT NULL,
    "created_by_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_agent_test_prompts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_test_prompts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_test_prompts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_test_prompts_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_test_prompts_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_test_prompts_prompt" CHECK (char_length("prompt") BETWEEN 1 AND 2000)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_test_prompts_agent" ON "agent_test_prompts"("organization_id", "business_unit_id", "agent_definition_id", "created_at");

--bun:split
COMMENT ON TABLE "agent_test_prompts" IS 'Prompts kept with an agent for trying it again after an edit';

--bun:split
SELECT trenova_rls.reconcile();
