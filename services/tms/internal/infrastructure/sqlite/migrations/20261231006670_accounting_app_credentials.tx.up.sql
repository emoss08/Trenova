-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006670_accounting_app_credentials.tx.up.sql

CREATE TABLE IF NOT EXISTS "accounting_app_credentials"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "integration_type" TEXT NOT NULL,
    "environment" TEXT NOT NULL,
    "client_id" TEXT NOT NULL,
    "client_secret_ciphertext" TEXT NOT NULL,
    "webhook_verifier_ciphertext" TEXT,
    "fingerprint" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_app_credentials" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_app_credentials_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_app_credentials_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_app_credentials_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_app_credentials_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline')),
    CONSTRAINT "ck_accounting_app_credentials_environment" CHECK ("environment" IN ('Sandbox', 'Production'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_app_credentials_tenant_type" ON "accounting_app_credentials" ("organization_id", "business_unit_id", "integration_type");

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "app_source" TEXT NOT NULL DEFAULT 'Instance';

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "app_environment" TEXT;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "app_fingerprint" TEXT;
