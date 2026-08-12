ALTER TABLE "carrier_invoice_matches"
    ADD COLUMN "matched_via" VARCHAR(50) NOT NULL DEFAULT 'Manual';

--bun:split
COMMENT ON COLUMN "carrier_invoice_matches"."matched_via" IS 'How the match was created: Manual for dispatcher-created matches, Auto for matches created by the inbound EDI 210 auto-match sweep';
