ALTER TABLE "carrier_settlement_controls"
    DROP COLUMN IF EXISTS "auto_match_inbound_invoices",
    DROP COLUMN IF EXISTS "auto_accept_within_tolerance";
