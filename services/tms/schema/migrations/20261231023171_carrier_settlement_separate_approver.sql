-- Whoever submits a carrier settlement cannot also approve it, as for driver
-- settlements. On by default; a one-person office turns it off in Carrier
-- Settlement Control.
ALTER TABLE "carrier_settlement_controls"
    ADD COLUMN IF NOT EXISTS "require_separate_approver" BOOLEAN NOT NULL DEFAULT TRUE;
