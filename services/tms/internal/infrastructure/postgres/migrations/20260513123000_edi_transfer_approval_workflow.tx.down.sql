DROP INDEX IF EXISTS "idx_edi_load_tender_transfers_approval_workflow";

ALTER TABLE "edi_load_tender_transfers"
    DROP COLUMN IF EXISTS "processed_at",
    DROP COLUMN IF EXISTS "processing_started_at",
    DROP COLUMN IF EXISTS "approval_workflow_run_id",
    DROP COLUMN IF EXISTS "approval_workflow_id";
