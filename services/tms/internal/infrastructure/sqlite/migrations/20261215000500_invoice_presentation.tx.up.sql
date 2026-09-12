-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000500_invoice_presentation.tx.up.sql

ALTER TABLE "invoices" ADD COLUMN "detail" TEXT NOT NULL DEFAULT 'Detailed';

--bun:split

ALTER TABLE "invoices" ADD COLUMN "section_by" TEXT NOT NULL DEFAULT 'Shipment';
