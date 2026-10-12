-- Business Central is the third accounting system. Connections and app
-- credentials accept it, and its webhooks are subscriptions Trenova creates,
-- renews every few days and deletes, so each one is recorded with the secret
-- Business Central echoes back on every notification, sealed like a token.
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_integration_type";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero', 'BusinessCentral'));

--bun:split
ALTER TABLE "accounting_app_credentials"
    DROP CONSTRAINT IF EXISTS "ck_accounting_app_credentials_integration_type";

--bun:split
ALTER TABLE "accounting_app_credentials"
    ADD CONSTRAINT "ck_accounting_app_credentials_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero', 'BusinessCentral'));

--bun:split
CREATE TABLE IF NOT EXISTS "accounting_webhook_subscriptions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "connection_id" varchar(100) NOT NULL,
    "integration_type" varchar(50) NOT NULL,
    "resource" varchar(100) NOT NULL,
    "external_subscription_id" varchar(100),
    "notification_url" text,
    "client_state_ciphertext" text,
    "etag" varchar(200),
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "expires_at" bigint,
    "last_attempt_at" bigint,
    "last_error" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_webhook_subscriptions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_webhook_subscriptions_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_webhook_subscriptions_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_webhook_subscriptions_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_accounting_webhook_subscriptions_integration_type" CHECK ("integration_type" IN ('BusinessCentral')),
    CONSTRAINT "ck_accounting_webhook_subscriptions_status" CHECK ("status" IN ('Pending', 'Active', 'Failed')),
    CONSTRAINT "ck_accounting_webhook_subscriptions_active" CHECK ("status" <> 'Active' OR ("external_subscription_id" IS NOT NULL AND "client_state_ciphertext" IS NOT NULL AND "expires_at" IS NOT NULL))
);

--bun:split
-- One subscription per resource per connection.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_webhook_subscriptions_resource" ON "accounting_webhook_subscriptions"("organization_id", "business_unit_id", "connection_id", "resource");

--bun:split
-- Notifications name only the subscription, so the webhook finds its row by
-- the provider's id before it knows the tenant.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_webhook_subscriptions_external" ON "accounting_webhook_subscriptions"("integration_type", "external_subscription_id") WHERE "external_subscription_id" IS NOT NULL;

--bun:split
-- The renewal job reads the subscriptions closest to expiry first.
CREATE INDEX IF NOT EXISTS "idx_accounting_webhook_subscriptions_expiry" ON "accounting_webhook_subscriptions"("status", "expires_at");

--bun:split
COMMENT ON TABLE "accounting_webhook_subscriptions" IS 'Webhook subscriptions Trenova holds at an accounting system that requires them to be created and renewed; client_state_ciphertext seals the secret each notification carries';

--bun:split
SELECT trenova_rls.reconcile();
