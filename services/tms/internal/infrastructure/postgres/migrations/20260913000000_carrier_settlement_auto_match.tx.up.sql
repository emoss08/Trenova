ALTER TABLE "carrier_settlement_controls"
    ADD COLUMN "auto_match_inbound_invoices" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN "auto_accept_within_tolerance" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN "carrier_settlement_controls"."auto_match_inbound_invoices" IS 'When on, inbound EDI 210 invoices that resolve to an accepted tender offer are matched to their carrier assignment automatically';

--bun:split
COMMENT ON COLUMN "carrier_settlement_controls"."auto_accept_within_tolerance" IS 'When on (and auto-match is on), auto-created matches within the variance tolerance are resolved into the settlement pool without review';
