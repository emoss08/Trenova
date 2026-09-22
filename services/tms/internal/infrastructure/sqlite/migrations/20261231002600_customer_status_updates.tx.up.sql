-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231002600_customer_status_updates.tx.up.sql

ALTER TABLE "customers" ADD COLUMN "status_update_preference" TEXT NOT NULL DEFAULT 'None';

--bun:split

ALTER TABLE "customers" ADD COLUMN "status_update_recipients" TEXT;
