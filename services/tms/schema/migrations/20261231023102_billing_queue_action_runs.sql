ALTER TABLE "billingqueue_approval_runs"
    ADD COLUMN IF NOT EXISTS "action" varchar(30) NOT NULL DEFAULT 'Approve',
    ADD COLUMN IF NOT EXISTS "params" jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE "billingqueue_approval_runs" RENAME COLUMN "approved_count" TO "succeeded_count";

ALTER TABLE "billingqueue_approval_runs"
    ADD CONSTRAINT "ck_billingqueue_approval_runs_action" CHECK ("action" IN ('Approve', 'Post', 'Hold', 'Assign', 'Cancel', 'RequestPaperwork', 'ResolveIssue'));

ALTER TABLE "billingqueue_approval_run_items"
    DROP CONSTRAINT IF EXISTS "ck_billingqueue_approval_run_items_status",
    DROP CONSTRAINT IF EXISTS "ck_billingqueue_approval_run_items_failure_code",
    DROP CONSTRAINT IF EXISTS "ck_billingqueue_approval_run_items_failure_pairing";

UPDATE
    "billingqueue_approval_run_items"
SET
    "status" = 'Succeeded'
WHERE
    "status" = 'Approved';

ALTER TABLE "billingqueue_approval_run_items"
    ADD CONSTRAINT "ck_billingqueue_approval_run_items_status" CHECK ("status" IN ('Pending', 'Succeeded', 'Failed', 'Skipped')),
    ADD CONSTRAINT "ck_billingqueue_approval_run_items_failure_code" CHECK ("failure_code" IS NULL OR "failure_code" IN ('NotReady', 'AlreadyDone', 'OnHold', 'Undone', 'Unexpected', 'NotApplicable')),
    ADD CONSTRAINT "ck_billingqueue_approval_run_items_failure_pairing" CHECK ("status" IN ('Pending', 'Succeeded') OR "failure_code" IS NOT NULL);
