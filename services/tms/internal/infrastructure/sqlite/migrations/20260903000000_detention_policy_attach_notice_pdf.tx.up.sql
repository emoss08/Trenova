-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260903000000_detention_policy_attach_notice_pdf.tx.up.sql

ALTER TABLE "detention_policies" ADD COLUMN "attach_notice_pdf" INTEGER NOT NULL DEFAULT 0;
