DROP TABLE IF EXISTS "accounting_webhook_subscriptions";

--bun:split
DELETE FROM "accounting_app_credentials" WHERE "integration_type" = 'BusinessCentral';

--bun:split
ALTER TABLE "accounting_app_credentials"
    DROP CONSTRAINT IF EXISTS "ck_accounting_app_credentials_integration_type";

--bun:split
ALTER TABLE "accounting_app_credentials"
    ADD CONSTRAINT "ck_accounting_app_credentials_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero'));

--bun:split
DELETE FROM "accounting_connections" WHERE "integration_type" = 'BusinessCentral';

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_integration_type";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero'));
