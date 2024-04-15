-- Modify "billing_controls" table
ALTER TABLE "billing_controls"
ADD COLUMN "enforce_customer_billing" boolean NOT NULL DEFAULT false;