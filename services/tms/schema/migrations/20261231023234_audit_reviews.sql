CREATE TABLE IF NOT EXISTS "audit_reviews"(
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "kind" VARCHAR(50) NOT NULL,
    "dedupe_key" VARCHAR(255) NOT NULL,
    "entry_id" VARCHAR(100) NOT NULL,
    "action" VARCHAR(150) NOT NULL,
    "resource" VARCHAR(50) NOT NULL,
    "resource_id" VARCHAR(100),
    "entity_ref" VARCHAR(255),
    "principal_id" VARCHAR(100) NOT NULL,
    "principal_type" VARCHAR(20) NOT NULL,
    "actor_name" VARCHAR(255),
    "count" BIGINT NOT NULL DEFAULT 1,
    "threshold" INTEGER NOT NULL DEFAULT 1,
    "window_minutes" INTEGER NOT NULL DEFAULT 1,
    "occurred_at" BIGINT NOT NULL,
    "reviewed_by_id" VARCHAR(100),
    "reviewed_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_audit_reviews" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_audit_reviews_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_reviews_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_audit_reviews_kind" CHECK ("kind" IN (
        'RoleChanged', 'APIKeyChanged', 'MFAReset', 'IdentityConfigChanged', 'CrossTenantRefused',
        'MalwareRefused', 'LegalHoldChanged', 'BulkDelete', 'BulkExport'
    )),
    CONSTRAINT "ck_audit_reviews_count" CHECK ("count" >= 1 AND "threshold" >= 1 AND "window_minutes" >= 1),
    CONSTRAINT "ck_audit_reviews_reviewed" CHECK (("reviewed_at" IS NULL) = ("reviewed_by_id" IS NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_audit_reviews_dedupe" ON "audit_reviews"("organization_id", "business_unit_id", "dedupe_key");

CREATE INDEX IF NOT EXISTS "idx_audit_reviews_tenant_open" ON "audit_reviews"("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC) WHERE "reviewed_at" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_audit_reviews_tenant_reviewed" ON "audit_reviews"("organization_id", "business_unit_id", "reviewed_at" DESC, "id" DESC) WHERE "reviewed_at" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_audit_reviews_tenant_entry" ON "audit_reviews"("organization_id", "business_unit_id", "entry_id");

COMMENT ON TABLE "audit_reviews" IS 'Each hit of an audit alert rule, kept until a security reviewer marks it reviewed';
