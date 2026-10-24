-- Billing control can transfer ready freight on its own schedule. Those runs
-- are recorded like any other transfer, under their own scope, and the sweep
-- reads the latest one to keep each organization to its chosen cadence.
ALTER TABLE "billing_transfer_runs"
    DROP CONSTRAINT IF EXISTS "chk_billing_transfer_runs_scope";

ALTER TABLE "billing_transfer_runs"
    ADD CONSTRAINT "chk_billing_transfer_runs_scope"
    CHECK ("scope" IN ('Selected', 'AllMatching', 'Retry', 'Automatic'));

CREATE INDEX IF NOT EXISTS "idx_billing_transfer_runs_tenant_scope_queued"
    ON "billing_transfer_runs" ("organization_id", "business_unit_id", "scope", "queued_at" DESC);
