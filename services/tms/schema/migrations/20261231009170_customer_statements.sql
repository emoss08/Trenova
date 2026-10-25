-- Customer statements: a PDF of a customer's account emailed to it, by hand
-- from the statement page or each month for the month just closed.
--
-- billing_controls carries the organization's monthly switch and the day of
-- the month statements go out; customer_billing_profiles lets one customer be
-- left out. Every statement sent is recorded, so a month is sent once and the
-- history shows what the customer was told and when.
ALTER TABLE "billing_controls"
    ADD COLUMN IF NOT EXISTS "monthly_statements_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "monthly_statement_day" SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE "billing_controls"
    DROP CONSTRAINT IF EXISTS "ck_billing_controls_monthly_statement_day";

ALTER TABLE "billing_controls"
    ADD CONSTRAINT "ck_billing_controls_monthly_statement_day"
        CHECK ("monthly_statement_day" BETWEEN 1 AND 28);

ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "send_statements" BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS "customer_statements" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "customer_id" VARCHAR(100) NOT NULL,
    "period_start" BIGINT NOT NULL DEFAULT 0,
    "period_end" BIGINT NOT NULL,
    "ending_balance_minor" BIGINT NOT NULL,
    "past_due_minor" BIGINT NOT NULL DEFAULT 0,
    "currency_code" VARCHAR(3) NOT NULL,
    "sent_to" TEXT NOT NULL,
    "monthly" BOOLEAN NOT NULL DEFAULT FALSE,
    "sent_by_id" VARCHAR(100),
    "sent_at" BIGINT NOT NULL,
    CONSTRAINT "pk_customer_statements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_customer_statements_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id")
        REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_customer_statements_sent_by" FOREIGN KEY ("sent_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "ck_customer_statements_period" CHECK ("period_start" >= 0 AND "period_end" >= "period_start")
);

-- A month's statement is sent to a customer once. Statements sent by hand
-- may repeat.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_customer_statements_monthly"
    ON "customer_statements"("organization_id", "business_unit_id", "customer_id", "period_end")
    WHERE "monthly";

CREATE INDEX IF NOT EXISTS "idx_customer_statements_customer"
    ON "customer_statements"("organization_id", "business_unit_id", "customer_id", "sent_at" DESC);

SELECT trenova_rls.reconcile();
