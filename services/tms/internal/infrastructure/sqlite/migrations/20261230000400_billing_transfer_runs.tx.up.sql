-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261230000400_billing_transfer_runs.tx.up.sql

CREATE TABLE IF NOT EXISTS "billing_transfer_runs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "requested_by_id" TEXT NOT NULL,
    "source_run_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Queued',
    "scope" TEXT NOT NULL,
    "search_query" TEXT,
    "shipment_status" TEXT,
    "bill_type" TEXT NOT NULL DEFAULT 'Invoice',
    "mark_completed_ready_to_invoice" INTEGER NOT NULL DEFAULT 0,
    "total_count" INTEGER NOT NULL DEFAULT 0,
    "processed_count" INTEGER NOT NULL DEFAULT 0,
    "transferred_count" INTEGER NOT NULL DEFAULT 0,
    "not_transferred_count" INTEGER NOT NULL DEFAULT 0,
    "skipped_count" INTEGER NOT NULL DEFAULT 0,
    "marked_ready_to_invoice_count" INTEGER NOT NULL DEFAULT 0,
    "retryable_count" INTEGER NOT NULL DEFAULT 0,
    "unmatched_count" INTEGER NOT NULL DEFAULT 0,
    "failure_message" TEXT,
    "cancel_requested_at" INTEGER,
    "cancel_requested_by_id" TEXT,
    "temporal_workflow_id" TEXT,
    "temporal_run_id" TEXT,
    "queued_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_billing_transfer_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billing_transfer_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_runs_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_billing_transfer_runs_status" CHECK ("status" IN ('Queued', 'Running', 'Completed', 'Canceled', 'Failed')),
    CONSTRAINT "chk_billing_transfer_runs_retry_source" CHECK ("scope" <> 'Retry' OR "source_run_id" IS NOT NULL),
    CONSTRAINT "chk_billing_transfer_runs_total_count" CHECK ("total_count" BETWEEN 0 AND 5000),
    CONSTRAINT "chk_billing_transfer_runs_counts_nonnegative" CHECK ("processed_count" >= 0 AND "transferred_count" >= 0 AND "not_transferred_count" >= 0 AND "skipped_count" >= 0 AND "marked_ready_to_invoice_count" >= 0 AND "retryable_count" >= 0 AND "unmatched_count" >= 0),
    CONSTRAINT "chk_billing_transfer_runs_cancel_pair" CHECK (("cancel_requested_at" IS NULL) = ("cancel_requested_by_id" IS NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_transfer_runs_active_requester" ON "billing_transfer_runs" ("organization_id", "business_unit_id", "requested_by_id")WHERE
    "status" IN ('Queued', 'Running');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_transfer_runs_tenant_created" ON "billing_transfer_runs" ("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_transfer_runs_stale" ON "billing_transfer_runs" ("status", "updated_at");

--bun:split

CREATE TABLE IF NOT EXISTS "billing_transfer_run_items"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "run_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "sequence" INTEGER NOT NULL,
    "pro_number" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "failure_code" TEXT,
    "error_message" TEXT,
    "marked_ready_to_invoice" INTEGER NOT NULL DEFAULT 0,
    "billing_queue_item_id" TEXT,
    "billing_queue_number" TEXT,
    "billing_queue_status" TEXT,
    "missing_requirements" TEXT NOT NULL DEFAULT '[]',
    "validation_failures" TEXT NOT NULL DEFAULT '[]',
    "processed_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_billing_transfer_run_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billing_transfer_run_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_run_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_run_items_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "billing_transfer_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_run_items_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_billing_transfer_run_items_status" CHECK ("status" IN ('Pending', 'Transferred', 'NotTransferred', 'Skipped')),
    CONSTRAINT "chk_billing_transfer_run_items_failure_code" CHECK ("failure_code" IS NULL OR "failure_code" IN ('NotFound', 'InvalidStatus', 'AlreadyTransferred', 'RequirementsUnmet', 'RateValidation', 'ReturnToOperations', 'Unexpected')),
    CONSTRAINT "chk_billing_transfer_run_items_sequence" CHECK ("sequence" >= 0),
    CONSTRAINT "chk_billing_transfer_run_items_failure_pairing" CHECK ("status" <> 'NotTransferred' OR "failure_code" IS NOT NULL),
    CONSTRAINT "chk_billing_transfer_run_items_processed" CHECK ("status" = 'Pending' OR "processed_at" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_transfer_run_items_run_shipment" ON "billing_transfer_run_items" ("run_id", "shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_transfer_run_items_run_sequence" ON "billing_transfer_run_items" ("run_id", "sequence");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_billing_transfer_run_items_run_status" ON "billing_transfer_run_items" ("run_id", "status");
