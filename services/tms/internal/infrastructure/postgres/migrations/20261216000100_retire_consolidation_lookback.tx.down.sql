--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- Restores the column at its original default. The per-customer values that were
-- dropped are not recoverable, and nothing reads them.
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "consolidation_lookback_days" smallint NOT NULL DEFAULT 30;
