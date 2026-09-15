-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: SQLite has no
-- ALTER TABLE ADD CONSTRAINT, so the status check and the message foreign key
-- are enforced by the application here. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261223000500_invoice_edi_send.tx.up.sql

ALTER TABLE "customer_billing_profiles" ADD COLUMN "email_invoice_enabled" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "edi_invoice_enabled" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "edi_send_status" TEXT NOT NULL DEFAULT 'NotSent';

--bun:split

ALTER TABLE "invoices" ADD COLUMN "last_edi_message_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "edi_sent_at" INTEGER;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "last_edi_error" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_edi_send_status" ON "invoices" ("organization_id", "business_unit_id", "edi_send_status")WHERE
    "edi_send_status" <> 'NotSent';

--bun:split

ALTER TABLE "edi_messages" ADD COLUMN "invoice_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_messages_invoice" ON "edi_messages" ("invoice_id", "organization_id", "business_unit_id")WHERE
    "invoice_id" IS NOT NULL;
