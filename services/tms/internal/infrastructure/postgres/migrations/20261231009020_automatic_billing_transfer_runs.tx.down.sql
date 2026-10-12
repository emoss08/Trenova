DROP INDEX IF EXISTS "idx_billing_transfer_runs_tenant_scope_queued";

--bun:split
DELETE FROM "billing_transfer_runs" WHERE "scope" = 'Automatic';

--bun:split
ALTER TABLE "billing_transfer_runs"
    DROP CONSTRAINT IF EXISTS "chk_billing_transfer_runs_scope";

--bun:split
ALTER TABLE "billing_transfer_runs"
    ADD CONSTRAINT "chk_billing_transfer_runs_scope"
    CHECK ("scope" IN ('Selected', 'AllMatching', 'Retry'));
