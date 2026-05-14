ALTER TYPE edi_load_tender_transfer_status_enum ADD VALUE IF NOT EXISTS 'Processing';

--bun:split
ALTER TABLE "edi_load_tender_transfers"
    ADD COLUMN IF NOT EXISTS "approval_workflow_id" varchar(255),
    ADD COLUMN IF NOT EXISTS "approval_workflow_run_id" varchar(255),
    ADD COLUMN IF NOT EXISTS "processing_started_at" bigint,
    ADD COLUMN IF NOT EXISTS "processed_at" bigint;

CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_approval_workflow"
    ON "edi_load_tender_transfers"("approval_workflow_id")
    WHERE "approval_workflow_id" IS NOT NULL;
