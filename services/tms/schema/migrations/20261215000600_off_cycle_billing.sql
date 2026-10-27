--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- Billing a statement customer outside their agreed cadence is allowed, but it is
-- never silent. Both columns hold the reason the biller gave, so a customer with
-- two invoices in a month that should have produced one can be explained without
-- reading the audit log.
--
-- On the run: a whole statement period billed before its boundary closed.
ALTER TABLE "invoice_runs"
    ADD COLUMN IF NOT EXISTS "off_cycle_reason" TEXT;

-- On the invoice: one shipment invoiced on its own for a customer whose freight
-- was supposed to accumulate onto a statement.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "off_cycle_reason" TEXT;
