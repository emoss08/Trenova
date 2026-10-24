--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- The customer's minimum and auto-bill settings are copied onto the group when a
-- run is built rather than re-read at commit, so a proposal an operator approved
-- cannot change under them because somebody edited the customer in between.
ALTER TABLE "invoice_run_groups"
    ADD COLUMN IF NOT EXISTS "minimum_amount" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "auto_bill" boolean NOT NULL DEFAULT FALSE;
