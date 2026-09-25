-- An organization can connect through its own Intuit app instead of the one
-- this instance is configured with, which is the only option on an instance
-- that has none. The secrets are sealed by the application before they are
-- written here.
CREATE TABLE IF NOT EXISTS "accounting_app_credentials"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "integration_type" varchar(50) NOT NULL,
    "environment" varchar(20) NOT NULL,
    "client_id" varchar(255) NOT NULL,
    "client_secret_ciphertext" text NOT NULL,
    "webhook_verifier_ciphertext" text,
    "fingerprint" varchar(64) NOT NULL,
    "updated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_app_credentials" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_app_credentials_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_app_credentials_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_app_credentials_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_app_credentials_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline')),
    CONSTRAINT "ck_accounting_app_credentials_environment" CHECK ("environment" IN ('Sandbox', 'Production'))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_app_credentials_tenant_type" ON "accounting_app_credentials"("organization_id", "business_unit_id", "integration_type");

--bun:split
-- A connection's tokens belong to the app that issued them, so each connection
-- records which one that was. Connections made before this existed went
-- through the instance's app.
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "app_source" varchar(20) NOT NULL DEFAULT 'Instance',
    ADD COLUMN IF NOT EXISTS "app_environment" varchar(20),
    ADD COLUMN IF NOT EXISTS "app_fingerprint" varchar(64);

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_app_source";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_app_source" CHECK ("app_source" IN ('Instance', 'Tenant'));

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_app_environment";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_app_environment" CHECK ("app_environment" IS NULL OR "app_environment" IN ('Sandbox', 'Production'));
