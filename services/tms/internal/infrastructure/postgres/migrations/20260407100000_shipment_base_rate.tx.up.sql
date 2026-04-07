--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS "base_rate" NUMERIC(19,4) NOT NULL DEFAULT 0;

--bun:split
UPDATE shipments SET base_rate = freight_charge_amount WHERE base_rate = 0 AND freight_charge_amount > 0;
