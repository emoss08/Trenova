CREATE TABLE IF NOT EXISTS "audit_dlq_entries" (
    "id" VARCHAR(100) PRIMARY KEY,
    "original_entry_id" VARCHAR(100) NOT NULL,
    "entry_data" JSONB NOT NULL,
    "failure_time" BIGINT NOT NULL,
    "retry_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "next_retry_at" BIGINT,
    "status" VARCHAR(20) NOT NULL DEFAULT 'pending',
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::BIGINT,
    CONSTRAINT "chk_audit_dlq_status" CHECK (status IN ('pending', 'retrying', 'failed', 'recovered'))
);

CREATE INDEX "idx_audit_dlq_status_retry" ON "audit_dlq_entries"("status", "next_retry_at") WHERE status IN ('pending', 'retrying');
CREATE INDEX "idx_audit_dlq_organization" ON "audit_dlq_entries"("organization_id");
CREATE INDEX "idx_audit_dlq_created" ON "audit_dlq_entries"("created_at");
