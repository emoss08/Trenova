-- Payment reminders: an email to a customer once its invoices go past due,
-- on the days past due the organization schedules, listing what is overdue.
--
-- billing_controls carries the organization's switch and schedule;
-- customer_billing_profiles lets one customer be left out. Every reminder
-- step an invoice reached is recorded once, so a step is never sent twice
-- and the history shows what the customer was told and when.
ALTER TABLE "billing_controls"
    ADD COLUMN IF NOT EXISTS "payment_reminders_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "payment_reminder_days" INTEGER[] NOT NULL DEFAULT '{1,15,30}';

ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "send_payment_reminders" BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS "invoice_payment_reminders" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "invoice_id" VARCHAR(100) NOT NULL,
    "customer_id" VARCHAR(100) NOT NULL,
    "step" SMALLINT NOT NULL,
    "days_past_due" INTEGER NOT NULL,
    "open_amount_minor" BIGINT NOT NULL,
    "currency_code" VARCHAR(3) NOT NULL,
    "sent_to" TEXT NOT NULL,
    "manual" BOOLEAN NOT NULL DEFAULT FALSE,
    "sent_by_id" VARCHAR(100),
    "sent_at" BIGINT NOT NULL,
    CONSTRAINT "pk_invoice_payment_reminders" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_invoice_payment_reminders_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id")
        REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_payment_reminders_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id")
        REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_payment_reminders_sent_by" FOREIGN KEY ("sent_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "ck_invoice_payment_reminders_step" CHECK ("step" >= 0)
);

-- A scheduled step is sent at most once per invoice. Reminders sent by hand
-- are step 0 and may repeat.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_payment_reminders_step"
    ON "invoice_payment_reminders"("organization_id", "business_unit_id", "invoice_id", "step")
    WHERE "step" > 0;

CREATE INDEX IF NOT EXISTS "idx_invoice_payment_reminders_customer"
    ON "invoice_payment_reminders"("organization_id", "business_unit_id", "customer_id", "sent_at" DESC);

SELECT trenova_rls.reconcile();
