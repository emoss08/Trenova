-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260411152000_add_unapplied_cash_account.tx.up.sql

ALTER TABLE "accounting_controls" ADD COLUMN "default_unapplied_cash_account_id" TEXT;
