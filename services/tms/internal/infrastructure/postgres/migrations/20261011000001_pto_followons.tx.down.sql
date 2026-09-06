--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "org_holidays";

--bun:split
ALTER TABLE "workers"
    DROP COLUMN IF EXISTS "leave_type";

--bun:split
DELETE FROM "worker_pto_ledger"
WHERE "entry_type" IN ('Payout', 'Forfeiture');

--bun:split
ALTER TABLE "worker_pto_ledger"
    DROP CONSTRAINT IF EXISTS "chk_worker_pto_ledger_amount_sign";

--bun:split
ALTER TABLE "worker_pto_ledger"
    ADD CONSTRAINT "chk_worker_pto_ledger_amount_sign" CHECK (("entry_type" IN ('OpeningBalance', 'Accrual', 'Reversal', 'Carryover') AND "amount_days" > 0) OR ("entry_type" IN ('Usage', 'Expiry') AND "amount_days" < 0) OR "entry_type" = 'Adjustment');

--bun:split
UPDATE "pto_policy_rules"
SET "accrual_method" = 'Monthly'
WHERE "accrual_method" = 'PerPayPeriod';

--bun:split
ALTER TABLE "pto_policy_rules"
    DROP COLUMN IF EXISTS "tiers",
    DROP COLUMN IF EXISTS "on_termination";

--bun:split
DROP TYPE IF EXISTS "org_holiday_kind_enum";

--bun:split
DROP TYPE IF EXISTS "worker_leave_type_enum";

--bun:split
DROP TYPE IF EXISTS "pto_termination_action_enum";
