--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- Statements no longer stop looking for owed freight after a fixed number of
-- days. The lookback's only effect was to exclude approved, unbilled freight
-- older than the window, and that included freight held under a customer's
-- minimum "for the next period" — which, once it aged past the window, was never
-- billed and appeared on no statement. With the exclusion gone the setting does
-- nothing, so it is removed rather than left as a control with no effect.
ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "consolidation_lookback_days";
