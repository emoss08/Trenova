-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: SQLite has no enum
-- types, so only the columns are dropped. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261225000000_remove_adjustment_supporting_document_requirement.tx.up.sql

ALTER TABLE "customer_billing_profiles" DROP COLUMN "invoice_adjustment_supporting_document_policy";

--bun:split

ALTER TABLE "invoice_adjustment_controls" DROP COLUMN "adjustment_attachment_requirement";
