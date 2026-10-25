-- Add partial composite index for PTO chart query optimization
CREATE INDEX IF NOT EXISTS idx_worker_pto_chart_approved
    ON "worker_pto"(
        "organization_id",
        "business_unit_id",
        "start_date",
        "end_date",
        "type",
        "worker_id"
    )
    WHERE "status" = 'Approved';

COMMENT ON INDEX idx_worker_pto_chart_approved IS 'Optimizes approved PTO chart queries by tenant, date window, type, and worker';
