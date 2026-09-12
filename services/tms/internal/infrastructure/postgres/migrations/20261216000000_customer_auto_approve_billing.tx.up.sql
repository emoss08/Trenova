--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- Lets freight that passes every billing requirement clear the billing queue
-- without a biller clicking Approve, so the queue holds only what needs a human.
--
-- Defaults to FALSE. Auto-approval is reachable only on the automatic transfer
-- path and only when the shipment has no requirement or rate issue at all, so
-- turning it on cannot put unverified freight in front of a customer.
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "auto_approve" BOOLEAN NOT NULL DEFAULT FALSE;
