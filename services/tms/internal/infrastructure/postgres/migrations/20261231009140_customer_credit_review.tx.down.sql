ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "credit_reviewed_at",
    DROP COLUMN IF EXISTS "credit_hold_automatic";
