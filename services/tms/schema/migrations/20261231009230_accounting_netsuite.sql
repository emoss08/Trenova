-- NetSuite is the fourth accounting system. Connections and app credentials
-- accept it. A NetSuite sign-in lasts a week, so a connection can also be made
-- with a certificate: Trenova generates the key pair, the administrator uploads
-- the certificate to NetSuite, and every access token comes from an assertion
-- signed with the key, sealed like a token. The key waits in
-- accounting_pending_certificates until the certificate is proven.
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_integration_type";

ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero', 'BusinessCentral', 'NetSuite'));

ALTER TABLE "accounting_app_credentials"
    DROP CONSTRAINT IF EXISTS "ck_accounting_app_credentials_integration_type";

ALTER TABLE "accounting_app_credentials"
    ADD CONSTRAINT "ck_accounting_app_credentials_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero', 'BusinessCentral', 'NetSuite'));

ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "auth_method" varchar(30) NOT NULL DEFAULT 'Authorization',
    ADD COLUMN IF NOT EXISTS "client_key_ciphertext" text,
    ADD COLUMN IF NOT EXISTS "client_certificate_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "client_certificate_expires_at" bigint;

ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_auth_method" CHECK ("auth_method" IN ('Authorization', 'ClientCredentials'));

-- A certificate connection always names its certificate. Its key is cleared,
-- like a token, when the connection is revoked or disconnected.
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_certificate" CHECK ("auth_method" <> 'ClientCredentials' OR ("client_certificate_id" IS NOT NULL AND "client_certificate_expires_at" IS NOT NULL));

CREATE TABLE IF NOT EXISTS "accounting_pending_certificates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "integration_type" varchar(50) NOT NULL,
    "private_key_ciphertext" text NOT NULL,
    "certificate_pem" text NOT NULL,
    "expires_at" bigint NOT NULL,
    "prepared_by_id" varchar(100) NOT NULL,
    "prepared_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_pending_certificates" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_pending_certificates_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_pending_certificates_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_pending_certificates_prepared_by" FOREIGN KEY ("prepared_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_accounting_pending_certificates_integration_type" CHECK ("integration_type" IN ('NetSuite'))
);

-- A tenant holds at most one certificate waiting to be uploaded per system;
-- preparing another replaces it.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_pending_certificates_type" ON "accounting_pending_certificates"("organization_id", "business_unit_id", "integration_type");

COMMENT ON TABLE "accounting_pending_certificates" IS 'A certificate Trenova generated for a certificate connection, waiting for the administrator to upload it to the accounting system; private_key_ciphertext seals its private key';

SELECT trenova_rls.reconcile();
