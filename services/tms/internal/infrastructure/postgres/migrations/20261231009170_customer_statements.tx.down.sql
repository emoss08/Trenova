DROP TABLE IF EXISTS "customer_statements";
ALTER TABLE "customer_billing_profiles" DROP COLUMN IF EXISTS "send_statements";
ALTER TABLE "billing_controls"
    DROP CONSTRAINT IF EXISTS "ck_billing_controls_monthly_statement_day",
    DROP COLUMN IF EXISTS "monthly_statement_day",
    DROP COLUMN IF EXISTS "monthly_statements_enabled";
