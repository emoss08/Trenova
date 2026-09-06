--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "worker_pto"
    ADD COLUMN IF NOT EXISTS "rejection_reason" varchar(255),
    ADD COLUMN IF NOT EXISTS "cancellation_reason" varchar(255),
    ADD COLUMN IF NOT EXISTS "cancelled_by_id" varchar(100);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_pto_worker_status_dates"
    ON "worker_pto"("organization_id", "business_unit_id", "worker_id", "status", "start_date", "end_date");
