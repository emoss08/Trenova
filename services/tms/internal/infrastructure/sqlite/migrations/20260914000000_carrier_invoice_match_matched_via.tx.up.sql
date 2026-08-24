-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260914000000_carrier_invoice_match_matched_via.tx.up.sql

ALTER TABLE "carrier_invoice_matches" ADD COLUMN "matched_via" TEXT NOT NULL DEFAULT 'Manual';
