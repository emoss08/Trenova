DROP TABLE IF EXISTS "accounting_pending_certificates";

--bun:split
DELETE FROM "accounting_app_credentials" WHERE "integration_type" = 'NetSuite';

--bun:split
ALTER TABLE "accounting_app_credentials"
    DROP CONSTRAINT IF EXISTS "ck_accounting_app_credentials_integration_type";

--bun:split
ALTER TABLE "accounting_app_credentials"
    ADD CONSTRAINT "ck_accounting_app_credentials_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero', 'BusinessCentral'));

--bun:split
DELETE FROM "accounting_connections" WHERE "integration_type" = 'NetSuite';

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_certificate";

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_auth_method";

--bun:split
ALTER TABLE "accounting_connections"
    DROP COLUMN IF EXISTS "client_certificate_expires_at",
    DROP COLUMN IF EXISTS "client_certificate_id",
    DROP COLUMN IF EXISTS "client_key_ciphertext",
    DROP COLUMN IF EXISTS "auth_method";

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_integration_type";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero', 'BusinessCentral'));
