-- A shared server (managed.tenancy: shared) holds many companies, one organization each, for the
-- Trenova control plane. cloud_tenants is the server's bookkeeping about them: which control plane
-- instance each tenant is, the organization it became, and what the server last accepted and
-- applied of the control plane's entry for it. It is a global table read and written only by the
-- tenant lifecycle under system scope, like the scoped managed state it sits beside: it is the
-- control plane's metadata about tenants, never a tenant's own data, and it outlives a removed
-- tenant so the server can keep reporting it removed. Its columns name the organization as
-- tenant_organization_id so the tenant purge, which deletes by organization_id, never touches it.
CREATE TABLE IF NOT EXISTS "cloud_tenants"(
    "instance_id" varchar(36) NOT NULL,
    "tenant_organization_id" varchar(100),
    "tenant_business_unit_id" varchar(100),
    "admin_user_id" varchar(100),
    "state" varchar(20) NOT NULL,
    "applied_revision" bigint NOT NULL DEFAULT 0,
    "desired_revision" bigint NOT NULL,
    "desired_digest" varchar(64) NOT NULL,
    "desired" jsonb NOT NULL,
    "admin_invite_sha256" varchar(64),
    "last_archive_command_id" varchar(36),
    "last_archive_purpose" varchar(20),
    "last_archive_location" varchar(1024),
    "last_archive_size_bytes" bigint,
    "last_archive_sha256" varchar(64),
    "last_archive_taken_at" bigint,
    "refused_revision" bigint,
    "refused_code" varchar(20),
    "refused_reason" varchar(500),
    "removal_started_at" bigint,
    "removed_at" bigint,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_cloud_tenants" PRIMARY KEY ("instance_id"),
    CONSTRAINT "ck_cloud_tenants_state" CHECK ("state" IN ('applying', 'active', 'suspended', 'removing', 'removed')),
    CONSTRAINT "ck_cloud_tenants_made" CHECK (
        ("tenant_organization_id" IS NULL) = ("tenant_business_unit_id" IS NULL)
        AND ("state" IN ('applying', 'removed') OR "tenant_organization_id" IS NOT NULL)
    ),
    CONSTRAINT "ck_cloud_tenants_revisions" CHECK ("applied_revision" >= 0 AND "desired_revision" >= 1),
    CONSTRAINT "ck_cloud_tenants_desired" CHECK (jsonb_typeof("desired") = 'object'),
    CONSTRAINT "ck_cloud_tenants_refusal" CHECK (
        ("refused_revision" IS NULL) = ("refused_code" IS NULL)
        AND ("refused_revision" IS NULL) = ("refused_reason" IS NULL)
    )
);

-- The email sender tags a message with its tenant's instance, and the usage report pages the
-- active tenants by organization: both read a tenant by its organization.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_cloud_tenants_organization"
    ON "cloud_tenants"("tenant_organization_id")
    WHERE "tenant_organization_id" IS NOT NULL;

COMMENT ON TABLE "cloud_tenants" IS 'The companies a shared server holds for the Trenova control plane, one organization each, and what the server last applied for each';

COMMENT ON COLUMN "cloud_tenants"."instance_id" IS 'The control plane instance the tenant is';

COMMENT ON COLUMN "cloud_tenants"."tenant_organization_id" IS 'The organization the tenant became; null until it is made';

COMMENT ON COLUMN "cloud_tenants"."state" IS 'applying, active, suspended, removing (its data is being deleted) or removed';

COMMENT ON COLUMN "cloud_tenants"."desired" IS 'The control plane''s entry for the tenant last accepted, as it was sent';

COMMENT ON COLUMN "cloud_tenants"."refused_reason" IS 'Why the server will not apply refused_revision as it is written, in one line that names nobody';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('cloud_tenants', 'A shared server''s bookkeeping about the tenants the control plane placed on it; read and written only by the tenant lifecycle under system scope; no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";

-- What happened to each tenant, kept after the tenant and its own audit trail are removed. It
-- names the control plane instance and holds counts and reasons, never a name, address or record
-- of the tenant.
CREATE TABLE IF NOT EXISTS "cloud_tenant_events"(
    "id" varchar(100) NOT NULL,
    "instance_id" varchar(36) NOT NULL,
    "kind" varchar(20) NOT NULL,
    "revision" bigint NOT NULL,
    "detail" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "occurred_at" bigint NOT NULL,
    CONSTRAINT "pk_cloud_tenant_events" PRIMARY KEY ("id"),
    CONSTRAINT "ck_cloud_tenant_events_kind" CHECK ("kind" IN ('created', 'applied', 'refused', 'suspended', 'resumed', 'removing', 'removed')),
    CONSTRAINT "ck_cloud_tenant_events_detail" CHECK (jsonb_typeof("detail") = 'object')
);

CREATE INDEX IF NOT EXISTS "idx_cloud_tenant_events_instance"
    ON "cloud_tenant_events"("instance_id", "occurred_at" DESC, "id" DESC);

COMMENT ON TABLE "cloud_tenant_events" IS 'The lifecycle of each tenant of a shared server, kept after the tenant is removed; no tenant data';

INSERT INTO trenova_rls.global_tables("table_name", "reason")
VALUES ('cloud_tenant_events', 'The lifecycle ledger of a shared server''s tenants, kept after removal; counts and reasons only, no tenant data')
ON CONFLICT ("table_name") DO UPDATE SET "reason" = EXCLUDED."reason";
