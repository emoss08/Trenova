-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260513123000_edi_transfer_approval_workflow.tx.up.sql

ALTER TABLE "edi_load_tender_transfers" ADD COLUMN "approval_workflow_id" TEXT;

--bun:split

ALTER TABLE "edi_load_tender_transfers" ADD COLUMN "approval_workflow_run_id" TEXT;

--bun:split

ALTER TABLE "edi_load_tender_transfers" ADD COLUMN "processing_started_at" INTEGER;

--bun:split

ALTER TABLE "edi_load_tender_transfers" ADD COLUMN "processed_at" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_load_tender_transfers_approval_workflow"
    ON "edi_load_tender_transfers" ("approval_workflow_id")WHERE "approval_workflow_id" IS NOT NULL;
