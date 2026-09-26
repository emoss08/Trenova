-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006870_driver_settlement_paid_journal.tx.up.sql

ALTER TABLE "driver_settlements" ADD COLUMN "paid_journal_batch_id" TEXT;
