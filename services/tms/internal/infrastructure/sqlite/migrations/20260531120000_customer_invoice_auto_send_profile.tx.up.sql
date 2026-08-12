-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260531120000_customer_invoice_auto_send_profile.tx.up.sql

ALTER TABLE "customer_billing_profiles" ADD COLUMN "auto_send_invoice_on_generation" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "customer_billing_profiles" DROP COLUMN "summary_transmit_on_generation";

--bun:split

ALTER TABLE "customer_email_profiles" DROP COLUMN "send_invoice_on_generation";
