-- Invoices are always sent from the organization's Billing email profile, whose
-- sender is the address verified with the email provider. A per-customer From
-- address overrode it on every send and was the usual cause of an unverified
-- domain being refused, so it is removed.
DROP INDEX IF EXISTS "idx_customer_email_profiles_from_email";

--bun:split
ALTER TABLE "customer_email_profiles"
    DROP COLUMN IF EXISTS "from_email";
