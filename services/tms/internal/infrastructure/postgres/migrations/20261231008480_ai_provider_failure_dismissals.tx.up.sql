-- A person putting away the notice that a provider is failing, up to the
-- failure they saw; a newer failure brings the notice back.
CREATE TABLE IF NOT EXISTS "ai_provider_failure_dismissals"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "provider_id" varchar(100) NOT NULL,
    "failure_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_ai_provider_failure_dismissals" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_provider_failure_dismissals_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_provider_failure_dismissals_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_provider_failure_dismissals_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_provider_failure_dismissals_provider" FOREIGN KEY ("provider_id", "business_unit_id", "organization_id") REFERENCES "ai_providers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_provider_failure_dismissals_user_provider" ON "ai_provider_failure_dismissals"("organization_id", "business_unit_id", "user_id", "provider_id");

--bun:split
COMMENT ON TABLE "ai_provider_failure_dismissals" IS 'Who put away the notice that a provider is failing, up to which failure';

--bun:split
SELECT trenova_rls.reconcile();
