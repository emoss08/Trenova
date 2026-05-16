DROP INDEX IF EXISTS "idx_edi_load_tender_transfers_open_unique";

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_open_unique"
    ON "edi_load_tender_transfers"("source_shipment_id", "source_partner_id")
    WHERE "status" NOT IN ('Approved', 'Rejected', 'Expired', 'Canceled', 'Failed');
