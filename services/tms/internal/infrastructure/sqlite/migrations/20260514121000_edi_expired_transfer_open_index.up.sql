-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260514121000_edi_expired_transfer_open_index.up.sql

DROP INDEX IF EXISTS "idx_edi_load_tender_transfers_open_unique";

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_open_unique"
    ON "edi_load_tender_transfers" ("source_shipment_id", "source_partner_id")WHERE "status" NOT IN ('Approved', 'Rejected', 'Expired', 'Canceled', 'Failed');
