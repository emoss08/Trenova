-- Each save of an organization's AI settings that people edit together: the
-- organization-wide agent controls and each model provider. A save that loses
-- a race to another reads them to say who saved in between, when and what
-- they changed. A provider's API key is never part of a snapshot.
CREATE TABLE IF NOT EXISTS "ai_setting_versions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "version" bigint NOT NULL,
    "snapshot" jsonb NOT NULL,
    "author_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_ai_setting_versions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_ai_setting_versions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_setting_versions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_setting_versions_author" FOREIGN KEY ("author_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_ai_setting_versions_kind" CHECK ("kind" IN ('AgentControl', 'AIProvider'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_setting_versions_version" ON "ai_setting_versions"("organization_id", "business_unit_id", "kind", "subject_id", "version" DESC);

--bun:split
COMMENT ON TABLE "ai_setting_versions" IS 'Each save of the organization-wide agent controls and of each model provider, to explain a save conflict';

--bun:split
SELECT trenova_rls.reconcile();
