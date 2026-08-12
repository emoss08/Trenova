-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260913000000_carrier_settlement_auto_match.tx.up.sql

ALTER TABLE "carrier_settlement_controls" ADD COLUMN "auto_match_inbound_invoices" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "carrier_settlement_controls" ADD COLUMN "auto_accept_within_tolerance" INTEGER NOT NULL DEFAULT 0;
