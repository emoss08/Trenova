DROP TABLE IF EXISTS "invoice_payment_reminders";
ALTER TABLE "customer_billing_profiles" DROP COLUMN IF EXISTS "send_payment_reminders";
ALTER TABLE "billing_controls"
    DROP COLUMN IF EXISTS "payment_reminder_days",
    DROP COLUMN IF EXISTS "payment_reminders_enabled";
