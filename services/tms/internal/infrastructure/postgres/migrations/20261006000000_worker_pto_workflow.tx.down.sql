--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_worker_pto_worker_status_dates";

--bun:split
ALTER TABLE "worker_pto"
    DROP COLUMN IF EXISTS "rejection_reason",
    DROP COLUMN IF EXISTS "cancellation_reason",
    DROP COLUMN IF EXISTS "cancelled_by_id";
