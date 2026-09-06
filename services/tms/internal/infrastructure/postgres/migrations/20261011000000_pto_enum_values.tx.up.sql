--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- New enum values live in their own migration: Postgres refuses to use a value
-- added by ALTER TYPE inside the same transaction, and the follow-on migration
-- references them in CHECK constraints.
ALTER TYPE "pto_accrual_method_enum"
    ADD VALUE IF NOT EXISTS 'PerPayPeriod';

--bun:split
ALTER TYPE "pto_ledger_entry_type_enum"
    ADD VALUE IF NOT EXISTS 'Payout';

--bun:split
ALTER TYPE "pto_ledger_entry_type_enum"
    ADD VALUE IF NOT EXISTS 'Forfeiture';
