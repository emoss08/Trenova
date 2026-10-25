CREATE TABLE IF NOT EXISTS "accounting_briefs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "brief_date" varchar(10) NOT NULL,
    "generation" integer NOT NULL,
    "trigger" varchar(20) NOT NULL,
    "segments" jsonb NOT NULL DEFAULT '[]',
    "facts" jsonb NOT NULL DEFAULT '{}',
    "narrated" boolean NOT NULL DEFAULT FALSE,
    "model_identifier" varchar(255),
    "generated_at" bigint NOT NULL,
    "verified_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_briefs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_briefs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_briefs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_accounting_briefs_trigger" CHECK ("trigger" IN ('Scheduled', 'OnDemand', 'Refreshed')),
    CONSTRAINT "ck_accounting_briefs_generation" CHECK ("generation" >= 1),
    CONSTRAINT "ck_accounting_briefs_verified_at" CHECK ("verified_at" >= "generated_at")
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_briefs_generation" ON "accounting_briefs"("organization_id", "business_unit_id", "brief_date", "generation");

CREATE INDEX IF NOT EXISTS "idx_accounting_briefs_date" ON "accounting_briefs"("brief_date");

COMMENT ON TABLE "accounting_briefs" IS 'The accounting dashboard brief for an organization''s day, written before the workday and again when the receivables figures it cites no longer hold';

SELECT trenova_rls.reconcile();
