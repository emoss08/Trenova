--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_worker_pto_pending";

--bun:split
ALTER TABLE "worker_pto"
    DROP COLUMN IF EXISTS "days",
    DROP COLUMN IF EXISTS "balance_after_days",
    DROP COLUMN IF EXISTS "auto_approved";

--bun:split
DROP TABLE IF EXISTS "worker_pto_ledger";

--bun:split
DROP TABLE IF EXISTS "worker_pto_balances";

--bun:split
DROP TABLE IF EXISTS "worker_pto_policy_assignments";

--bun:split
DROP TABLE IF EXISTS "pto_policy_rules";

--bun:split
DROP TABLE IF EXISTS "pto_policies";

--bun:split
DROP TYPE IF EXISTS "pto_ledger_actor_type_enum";

--bun:split
DROP TYPE IF EXISTS "pto_ledger_entry_type_enum";

--bun:split
DROP TYPE IF EXISTS "pto_accrual_method_enum";

--bun:split
DROP TYPE IF EXISTS "pto_year_basis_enum";

--bun:split
DROP TYPE IF EXISTS "pto_policy_status_enum";
