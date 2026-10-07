-- Changes to how AI is set up that the last 30 days of runs argue for,
-- computed nightly per tenant and applied or put away by a person.
CREATE TABLE IF NOT EXISTS "ai_tune_ups"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "kind" varchar(40) NOT NULL,
    "fingerprint" varchar(400) NOT NULL,
    "agent_definition_id" varchar(100),
    "provider_id" varchar(100),
    "other_provider_id" varchar(100),
    "tool_name" varchar(100),
    "task" varchar(100),
    "evidence" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "status" varchar(20) NOT NULL,
    "dismissed_until" bigint,
    "decided_by_id" varchar(100),
    "decided_at" bigint,
    "computed_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_ai_tune_ups" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_tune_ups_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_tune_ups_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_tune_ups_decided_by" FOREIGN KEY ("decided_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_ai_tune_ups_kind" CHECK ("kind" IN ('RaiseToolTier', 'ReorderProviders', 'LeaveShadow', 'AssignTask', 'TurnOffIdleAgent')),
    CONSTRAINT "ck_ai_tune_ups_status" CHECK ("status" IN ('Open', 'Applied', 'Dismissed'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_tune_ups_fingerprint" ON "ai_tune_ups"("organization_id", "business_unit_id", "fingerprint");

--bun:split
COMMENT ON TABLE "ai_tune_ups" IS 'Suggested changes to AI setup drawn from recent runs, and what people decided about them';

--bun:split
SELECT trenova_rls.reconcile();
