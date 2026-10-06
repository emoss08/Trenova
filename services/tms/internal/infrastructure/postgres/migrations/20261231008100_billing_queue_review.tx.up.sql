-- Why an item is on hold, who held it and what it was before, so releasing the
-- hold puts it back where it was rather than always at the start of review.
ALTER TABLE "billing_queue_items"
    ADD COLUMN IF NOT EXISTS "hold_reason_code" varchar(30),
    ADD COLUMN IF NOT EXISTS "held_at" bigint,
    ADD COLUMN IF NOT EXISTS "held_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "status_before_hold" billing_queue_status;

--bun:split
ALTER TABLE "billing_queue_items"
    DROP CONSTRAINT IF EXISTS "ck_billing_queue_items_hold_reason_code";

--bun:split
ALTER TABLE "billing_queue_items"
    ADD CONSTRAINT "ck_billing_queue_items_hold_reason_code" CHECK ("hold_reason_code" IS NULL OR "hold_reason_code" IN ('WaitingOnPaperwork', 'CustomerDispute', 'RateQuestion'));

--bun:split
-- The queue's own order, and the neighbours lookup that steps through it.
CREATE INDEX IF NOT EXISTS "idx_billing_queue_items_tenant_created" ON "billing_queue_items"("organization_id", "business_unit_id", "created_at" DESC, "id" DESC);

--bun:split
-- The duplicate check reads every other item for the same shipment and payer.
CREATE INDEX IF NOT EXISTS "idx_billing_queue_items_shipment_payer" ON "billing_queue_items"("organization_id", "business_unit_id", "shipment_id", "bill_to_customer_id");

--bun:split
-- Whether a document is signed. Unknown until extraction or a person says so;
-- billing reads it for the documents whose type needs a signature.
ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS "signature_status" varchar(20) NOT NULL DEFAULT 'Unknown',
    ADD COLUMN IF NOT EXISTS "signed_at" bigint;

--bun:split
ALTER TABLE "documents"
    DROP CONSTRAINT IF EXISTS "ck_documents_signature_status";

--bun:split
ALTER TABLE "documents"
    ADD CONSTRAINT "ck_documents_signature_status" CHECK ("signature_status" IN ('Unknown', 'Signed', 'Unsigned'));

--bun:split
ALTER TABLE "document_types"
    ADD COLUMN IF NOT EXISTS "requires_signature" boolean NOT NULL DEFAULT FALSE;

--bun:split
-- A proof of delivery is only proof once the receiver has signed it.
UPDATE
    "document_types"
SET
    "requires_signature" = TRUE
WHERE
    "code" = 'POD';

--bun:split
CREATE TABLE IF NOT EXISTS "billing_queue_issues"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "item_id" varchar(100) NOT NULL,
    "check_key" varchar(20) NOT NULL,
    "code" varchar(50) NOT NULL,
    "subject_key" varchar(100) NOT NULL DEFAULT '',
    "summary" text NOT NULL,
    "reasoning" text,
    "source" varchar(20) NOT NULL DEFAULT 'Deterministic',
    "agent_run_id" varchar(100),
    "flagged_charge_id" varchar(100),
    "options" jsonb NOT NULL DEFAULT '[]',
    "resolution_key" varchar(30),
    "resolution_text" text,
    "effect_snapshot" jsonb,
    "resolved_by_id" varchar(100),
    "resolved_at" bigint,
    "requested_at" bigint,
    "requested_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_billing_queue_issues" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billing_queue_issues_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_issues_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_issues_item" FOREIGN KEY ("item_id", "business_unit_id", "organization_id") REFERENCES "billing_queue_items"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_billing_queue_issues_check_key" CHECK ("check_key" IN ('biller', 'charges', 'pod', 'terms', 'duplicate')),
    CONSTRAINT "ck_billing_queue_issues_source" CHECK ("source" IN ('Deterministic', 'Agent')),
    CONSTRAINT "ck_billing_queue_issues_resolution_pair" CHECK (("resolution_key" IS NULL) = ("resolved_at" IS NULL))
);

--bun:split
-- One issue per finding: the sync that raises them runs on every read and has
-- to land on the same row each time.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_queue_issues_finding" ON "billing_queue_issues"("organization_id", "business_unit_id", "item_id", "code", "subject_key");

--bun:split
-- What still needs a person, per item: the table's badge and the approval guard.
CREATE INDEX IF NOT EXISTS "idx_billing_queue_issues_open" ON "billing_queue_issues"("organization_id", "business_unit_id", "item_id")
WHERE
    "resolution_key" IS NULL;

--bun:split
COMMENT ON TABLE "billing_queue_issues" IS 'Something about a billing queue item that a person has to settle before it can be approved: a charge not on the rate con, an unsigned POD, detention that disagrees with the ELD. Raised deterministically; an agent may add reasoning.';

--bun:split
CREATE TABLE IF NOT EXISTS "billing_queue_events"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "item_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "text" text NOT NULL,
    "actor_type" varchar(10) NOT NULL,
    "actor_id" varchar(100),
    "actor_name" varchar(255),
    "payload" jsonb NOT NULL DEFAULT '{}',
    "at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_billing_queue_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billing_queue_events_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_events_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_events_item" FOREIGN KEY ("item_id", "business_unit_id", "organization_id") REFERENCES "billing_queue_items"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_billing_queue_events_actor_type" CHECK ("actor_type" IN ('System', 'Agent', 'User')),
    CONSTRAINT "ck_billing_queue_events_kind" CHECK ("kind" IN ('Transferred', 'Assigned', 'IssueRaised', 'IssueResolved', 'IssueUndone', 'IssueCleared', 'DocumentRequested', 'Held', 'Released', 'Approved', 'Posted', 'StatusChanged'))
);

--bun:split
-- The item's activity, newest first, a page at a time.
CREATE INDEX IF NOT EXISTS "idx_billing_queue_events_item_at" ON "billing_queue_events"("organization_id", "business_unit_id", "item_id", "at" DESC, "id" DESC);

--bun:split
COMMENT ON TABLE "billing_queue_events" IS 'What happened to a billing queue item, in plain words, and who did it: the system, an agent or a person.';

--bun:split
CREATE TABLE IF NOT EXISTS "billingqueue_approval_runs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "requested_by_id" varchar(100) NOT NULL,
    "idempotency_key" varchar(100) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Scheduled',
    "total_count" integer NOT NULL DEFAULT 0,
    "approved_count" integer NOT NULL DEFAULT 0,
    "failed_count" integer NOT NULL DEFAULT 0,
    "skipped_count" integer NOT NULL DEFAULT 0,
    "commit_at" bigint NOT NULL,
    "failure_message" text,
    "cancel_requested_at" bigint,
    "cancel_requested_by_id" varchar(100),
    "temporal_workflow_id" varchar(255),
    "temporal_run_id" varchar(255),
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_billingqueue_approval_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billingqueue_approval_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billingqueue_approval_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billingqueue_approval_runs_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_billingqueue_approval_runs_status" CHECK ("status" IN ('Scheduled', 'Running', 'Completed', 'Undone', 'Failed')),
    CONSTRAINT "ck_billingqueue_approval_runs_total" CHECK ("total_count" BETWEEN 0 AND 500),
    CONSTRAINT "ck_billingqueue_approval_runs_cancel_pair" CHECK (("cancel_requested_at" IS NULL) = ("cancel_requested_by_id" IS NULL))
);

--bun:split
-- The client sends one key per press of Approve; a retried request finds the
-- run it already started instead of approving the same items twice.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_billingqueue_approval_runs_idempotency" ON "billingqueue_approval_runs"("organization_id", "business_unit_id", "idempotency_key");

--bun:split
-- The reconciler sweeps runs that stopped moving.
CREATE INDEX IF NOT EXISTS "idx_billingqueue_approval_runs_stale" ON "billingqueue_approval_runs"("status", "updated_at");

--bun:split
COMMENT ON TABLE "billingqueue_approval_runs" IS 'One bulk approval of billing queue items, executed by a Temporal workflow that waits a few seconds for an undo before it touches anything.';

--bun:split
CREATE TABLE IF NOT EXISTS "billingqueue_approval_run_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "item_id" varchar(100) NOT NULL,
    "sequence" integer NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "failure_code" varchar(30),
    "error_message" text,
    "invoice_id" varchar(100),
    "invoice_number" varchar(100),
    "processed_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_billingqueue_approval_run_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_billingqueue_approval_run_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billingqueue_approval_run_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billingqueue_approval_run_items_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "billingqueue_approval_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_billingqueue_approval_run_items_item" FOREIGN KEY ("item_id", "business_unit_id", "organization_id") REFERENCES "billing_queue_items"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_billingqueue_approval_run_items_status" CHECK ("status" IN ('Pending', 'Approved', 'Failed', 'Skipped')),
    CONSTRAINT "ck_billingqueue_approval_run_items_failure_code" CHECK ("failure_code" IS NULL OR "failure_code" IN ('NotReady', 'AlreadyDone', 'OnHold', 'Undone', 'Unexpected')),
    CONSTRAINT "ck_billingqueue_approval_run_items_failure_pairing" CHECK ("status" IN ('Pending', 'Approved') OR "failure_code" IS NOT NULL)
);

--bun:split
-- An item appears once per run, which is what makes recording its outcome
-- idempotent when Temporal retries the activity that wrote it.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_billingqueue_approval_run_items_run_item" ON "billingqueue_approval_run_items"("run_id", "item_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_billingqueue_approval_run_items_run_sequence" ON "billingqueue_approval_run_items"("run_id", "sequence");

--bun:split
COMMENT ON TABLE "billingqueue_approval_run_items" IS 'One item inside a bulk approval run and what became of it: approved with its invoice, or why not.';
