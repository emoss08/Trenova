ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "summary_transmit_on_generation" boolean NOT NULL DEFAULT TRUE;

ALTER TABLE "customer_email_profiles"
    ADD COLUMN IF NOT EXISTS "send_invoice_on_generation" boolean NOT NULL DEFAULT TRUE;

UPDATE "customer_billing_profiles"
SET "summary_transmit_on_generation" = "auto_send_invoice_on_generation";

UPDATE "customer_email_profiles" AS cem
SET "send_invoice_on_generation" = cbp."auto_send_invoice_on_generation"
FROM "customer_billing_profiles" AS cbp
WHERE cbp."customer_id" = cem."customer_id"
  AND cbp."organization_id" = cem."organization_id"
  AND cbp."business_unit_id" = cem."business_unit_id";

ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "auto_send_invoice_on_generation";
