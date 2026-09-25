-- A tenant's link to its accounting system. The tokens rotate on their own
-- schedule, so they live here rather than in the integration's configuration,
-- which only an administrator's save writes.
CREATE TABLE IF NOT EXISTS "accounting_connections"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "integration_type" varchar(50) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Connected',
    "external_realm_id" varchar(100) NOT NULL,
    "external_company_name" varchar(255),
    "external_legal_name" varchar(255),
    "external_country" varchar(10),
    "external_home_currency" varchar(3),
    "external_multi_currency_enabled" boolean NOT NULL DEFAULT FALSE,
    "external_books_closed_through" bigint,
    "access_token_ciphertext" text,
    "access_token_expires_at" bigint NOT NULL DEFAULT 0,
    "refresh_token_ciphertext" text,
    "refresh_token_expires_at" bigint NOT NULL DEFAULT 0,
    "refresh_token_absolute_expires_at" bigint NOT NULL DEFAULT 0,
    "last_refreshed_at" bigint,
    "last_checked_at" bigint,
    "last_success_at" bigint,
    "last_failure_at" bigint,
    "consecutive_failures" integer NOT NULL DEFAULT 0,
    "last_error_category" varchar(30),
    "last_error_message" text,
    "last_webhook_at" bigint,
    "connected_by_id" varchar(100),
    "connected_at" bigint NOT NULL,
    "disconnected_by_id" varchar(100),
    "disconnected_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_connections_tenant_type" ON "accounting_connections"("organization_id", "business_unit_id", "integration_type");

--bun:split
-- Webhooks and documents are routed by the provider's company id, so one
-- company can only be connected to one tenant at a time.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_connections_realm" ON "accounting_connections"("integration_type", "external_realm_id") WHERE "status" <> 'Disconnected';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_connections_health" ON "accounting_connections"("status", "access_token_expires_at");
