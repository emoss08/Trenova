-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260409143000_add_customer_adjustment_supporting_document_policy.tx.up.sql

ALTER TABLE "customer_billing_profiles" ADD COLUMN "invoice_adjustment_supporting_document_policy" TEXT NOT NULL DEFAULT 'Inherit';
