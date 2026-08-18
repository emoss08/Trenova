-- A routing guide entry carries the rate somebody typed the day the guide was
-- written. An entry marked to use the contract instead offers what the carrier
-- actually negotiated, read at tender time.
--
-- The frozen rate stays as the fallback for a lane no contract covers, so
-- turning this on cannot leave an entry with no rate at all.
ALTER TABLE "routing_guide_entries"
    ADD COLUMN IF NOT EXISTS "use_contract_rate" BOOLEAN NOT NULL DEFAULT FALSE;
