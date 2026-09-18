CREATE TABLE IF NOT EXISTS "billing_transfer_runs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "requested_by_id" varchar(100) NOT NULL,
    "source_run_id" varchar(100),
    "status" varchar(20) NOT NULL DEFAULT 'Queued',
    "scope" varchar(20) NOT NULL,
    "search_query" varchar(200),
    "shipment_status" shipment_status_enum,
    "bill_type" billing_type NOT NULL DEFAULT 'Invoice',
    "mark_completed_ready_to_invoice" boolean NOT NULL DEFAULT FALSE,
    "total_count" integer NOT NULL DEFAULT 0,
    "processed_count" integer NOT NULL DEFAULT 0,
    "transferred_count" integer NOT NULL DEFAULT 0,
    "not_transferred_count" integer NOT NULL DEFAULT 0,
    "skipped_count" integer NOT NULL DEFAULT 0,
    "marked_ready_to_invoice_count" integer NOT NULL DEFAULT 0,
    "retryable_count" integer NOT NULL DEFAULT 0,
    "unmatched_count" integer NOT NULL DEFAULT 0,
    "failure_message" text,
    "cancel_requested_at" bigint,
    "cancel_requested_by_id" varchar(100),
    "temporal_workflow_id" varchar(255),
    "temporal_run_id" varchar(255),
    "queued_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_billing_transfer_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billing_transfer_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_runs_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_billing_transfer_runs_status" CHECK ("status" IN ('Queued', 'Running', 'Completed', 'Canceled', 'Failed')),
    CONSTRAINT "chk_billing_transfer_runs_scope" CHECK ("scope" IN ('Selected', 'AllMatching', 'Retry')),
    CONSTRAINT "chk_billing_transfer_runs_retry_source" CHECK ("scope" <> 'Retry' OR "source_run_id" IS NOT NULL),
    CONSTRAINT "chk_billing_transfer_runs_total_count" CHECK ("total_count" BETWEEN 0 AND 5000),
    CONSTRAINT "chk_billing_transfer_runs_counts_nonnegative" CHECK ("processed_count" >= 0 AND "transferred_count" >= 0 AND "not_transferred_count" >= 0 AND "skipped_count" >= 0 AND "marked_ready_to_invoice_count" >= 0 AND "retryable_count" >= 0 AND "unmatched_count" >= 0),
    CONSTRAINT "chk_billing_transfer_runs_cancel_pair" CHECK (("cancel_requested_at" IS NULL) = ("cancel_requested_by_id" IS NULL))
);

--bun:split
-- The reattach lookup, and the guard behind it: a user has at most one run in
-- flight, so a double-submit is rejected here rather than quietly starting a
-- second run over the same shipments.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_transfer_runs_active_requester" ON "billing_transfer_runs"("organization_id", "business_unit_id", "requested_by_id")
WHERE
    "status" IN ('Queued', 'Running');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_billing_transfer_runs_tenant_created" ON "billing_transfer_runs"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--bun:split
-- The zombie reconciler sweeps non-terminal runs that stopped being updated.
CREATE INDEX IF NOT EXISTS "idx_billing_transfer_runs_stale" ON "billing_transfer_runs"("status", "updated_at");

--bun:split
COMMENT ON TABLE "billing_transfer_runs" IS 'One bulk transfer of shipments into the billing queue, executed by a Temporal workflow. Counters are derived from billing_transfer_run_items rather than incremented, so an activity retry cannot double-count.';

--bun:split
CREATE TABLE IF NOT EXISTS "billing_transfer_run_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "sequence" integer NOT NULL,
    "pro_number" varchar(100),
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "failure_code" varchar(50),
    "error_message" text,
    "marked_ready_to_invoice" boolean NOT NULL DEFAULT FALSE,
    "billing_queue_item_id" varchar(100),
    "billing_queue_number" varchar(100),
    "billing_queue_status" billing_queue_status,
    "missing_requirements" jsonb NOT NULL DEFAULT '[]',
    "validation_failures" jsonb NOT NULL DEFAULT '[]',
    "processed_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_billing_transfer_run_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billing_transfer_run_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_run_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_run_items_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "billing_transfer_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_transfer_run_items_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_billing_transfer_run_items_status" CHECK ("status" IN ('Pending', 'Transferred', 'NotTransferred', 'Skipped')),
    CONSTRAINT "chk_billing_transfer_run_items_failure_code" CHECK ("failure_code" IS NULL OR "failure_code" IN ('NotFound', 'InvalidStatus', 'AlreadyTransferred', 'RequirementsUnmet', 'RateValidation', 'ReturnToOperations', 'Unexpected')),
    CONSTRAINT "chk_billing_transfer_run_items_sequence" CHECK ("sequence" >= 0),
    -- A shipment that did not transfer always says why; the report has no blanks.
    CONSTRAINT "chk_billing_transfer_run_items_failure_pairing" CHECK ("status" <> 'NotTransferred' OR "failure_code" IS NOT NULL),
    CONSTRAINT "chk_billing_transfer_run_items_processed" CHECK ("status" = 'Pending' OR "processed_at" IS NOT NULL)
);

--bun:split
-- A shipment appears at most once per run. This is what makes recording a batch's
-- outcomes idempotent when Temporal retries the activity that wrote them.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_transfer_run_items_run_shipment" ON "billing_transfer_run_items"("run_id", "shipment_id");

--bun:split
-- Claiming the next batch and paging the report both read in request order.
CREATE INDEX IF NOT EXISTS "idx_billing_transfer_run_items_run_sequence" ON "billing_transfer_run_items"("run_id", "sequence");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_billing_transfer_run_items_run_status" ON "billing_transfer_run_items"("run_id", "status");

--bun:split
COMMENT ON TABLE "billing_transfer_run_items" IS 'One shipment inside a billing transfer run: its outcome, why it did not transfer, and the readiness detail the biller has to act on. Rows are seeded Pending up front; whatever is still Pending when the run ends becomes Skipped.';
