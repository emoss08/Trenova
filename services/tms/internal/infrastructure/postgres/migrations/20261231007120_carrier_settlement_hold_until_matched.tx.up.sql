--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "carrier_settlement_controls"
    ADD COLUMN IF NOT EXISTS "hold_until_invoice_matched" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN "carrier_settlement_controls"."hold_until_invoice_matched" IS 'When on, a load''s carrier cost stays out of settlements until a carrier invoice match for its assignment is resolved, and a settlement holding unmatched cost cannot be approved';
