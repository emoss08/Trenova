-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed. Hand-edits are
-- preserved only if you stop regenerating this file; see docs/databases.md.
-- Source: 20261225000000_remove_adjustment_supporting_document_requirement.tx.down.sql

ALTER TABLE "invoice_adjustment_controls" ADD COLUMN "adjustment_attachment_requirement" TEXT NOT NULL DEFAULT 'Optional';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "invoice_adjustment_supporting_document_policy" TEXT NOT NULL DEFAULT 'Inherit';
