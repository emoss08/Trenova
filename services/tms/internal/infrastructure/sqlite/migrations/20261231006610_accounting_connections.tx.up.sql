-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006510_accounting_connections.tx.up.sql

CREATE TABLE IF NOT EXISTS "accounting_connections"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "integration_type" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Connected',
    "external_realm_id" TEXT NOT NULL,
    "external_company_name" TEXT,
    "external_legal_name" TEXT,
    "external_country" TEXT,
    "external_home_currency" TEXT,
    "external_multi_currency_enabled" INTEGER NOT NULL DEFAULT 0,
    "external_books_closed_through" INTEGER,
    "access_token_ciphertext" TEXT,
    "access_token_expires_at" INTEGER NOT NULL DEFAULT 0,
    "refresh_token_ciphertext" TEXT,
    "refresh_token_expires_at" INTEGER NOT NULL DEFAULT 0,
    "refresh_token_absolute_expires_at" INTEGER NOT NULL DEFAULT 0,
    "last_refreshed_at" INTEGER,
    "last_checked_at" INTEGER,
    "last_success_at" INTEGER,
    "last_failure_at" INTEGER,
    "consecutive_failures" INTEGER NOT NULL DEFAULT 0,
    "last_error_category" TEXT,
    "last_error_message" TEXT,
    "last_webhook_at" INTEGER,
    "connected_by_id" TEXT,
    "connected_at" INTEGER NOT NULL,
    "disconnected_by_id" TEXT,
    "disconnected_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_connections" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_connections_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_connections_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_connections_connected_by" FOREIGN KEY ("connected_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_accounting_connections_disconnected_by" FOREIGN KEY ("disconnected_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_connections_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline')),
    CONSTRAINT "ck_accounting_connections_status" CHECK ("status" IN ('Connected', 'Degraded', 'Failing', 'Revoked', 'Disconnected')),
    CONSTRAINT "ck_accounting_connections_last_error_category" CHECK ("last_error_category" IS NULL OR "last_error_category" IN ('Transient', 'RateLimited', 'Unauthorized', 'Revoked', 'Configuration', 'Unknown'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_connections_tenant_type" ON "accounting_connections" ("organization_id", "business_unit_id", "integration_type");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_connections_realm" ON "accounting_connections" ("integration_type", "external_realm_id")WHERE "status" <> 'Disconnected';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_connections_health" ON "accounting_connections" ("status", "access_token_expires_at");
