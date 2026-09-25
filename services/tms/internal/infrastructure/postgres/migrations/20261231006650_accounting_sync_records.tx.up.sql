-- When a connection starts sending documents, whether they send on their own,
-- and who paused it. Connections that finished mapping before the start date
-- step existed resume at it, so nothing is sent until a start date is chosen.
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "sync_start_date" bigint,
    ADD COLUMN IF NOT EXISTS "sync_enabled_at" bigint,
    ADD COLUMN IF NOT EXISTS "auto_sync" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "paused_at" bigint,
    ADD COLUMN IF NOT EXISTS "paused_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "paused_reason" text;

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_setup_step";

--bun:split
UPDATE "accounting_connections" SET "setup_step" = 'StartDate' WHERE "setup_step" = 'Complete' AND "sync_start_date" IS NULL;

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_setup_step" CHECK ("setup_step" IN ('Mappings', 'StartDate', 'Complete'));

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_sync_ready";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_sync_ready" CHECK ("setup_step" <> 'Complete' OR ("sync_start_date" IS NOT NULL AND "sync_enabled_at" IS NOT NULL));

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "fk_accounting_connections_paused_by";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "fk_accounting_connections_paused_by" FOREIGN KEY ("paused_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
-- The outbox and the ledger of every document sent to an accounting system:
-- one row per document, operation and revision per connection, written in the
-- same transaction as the document it sends.
CREATE TABLE IF NOT EXISTS "accounting_sync_records"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "connection_id" varchar(100) NOT NULL,
    "object_type" varchar(30) NOT NULL,
    "object_id" varchar(100) NOT NULL,
    "object_number" varchar(100),
    "operation" varchar(20) NOT NULL,
    "source_event" varchar(40) NOT NULL,
    "idempotency_key" varchar(200) NOT NULL,
    "request_id" varchar(50) NOT NULL,
    "revision" bigint NOT NULL DEFAULT 1,
    "document_date" bigint,
    "depends_on_record_id" varchar(100),
    "status" varchar(20) NOT NULL DEFAULT 'Queued',
    "attempt_count" integer NOT NULL DEFAULT 0,
    "next_attempt_at" bigint,
    "lease_expires_at" bigint,
    "external_id" varchar(100),
    "external_doc_number" varchar(100),
    "external_url" text,
    "external_refs" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "payload_hash" varchar(64),
    "payload" jsonb,
    "mapping_ids" text[] NOT NULL DEFAULT '{}',
    "error_category" varchar(30),
    "error_code" varchar(50),
    "error_message" text,
    "resolution" text,
    "queued_at" bigint NOT NULL,
    "started_at" bigint,
    "synced_at" bigint,
    "released_by_id" varchar(100),
    "skipped_by_id" varchar(100),
    "skipped_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_sync_records" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_sync_records_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_depends_on" FOREIGN KEY ("depends_on_record_id", "business_unit_id", "organization_id") REFERENCES "accounting_sync_records"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("depends_on_record_id"),
    CONSTRAINT "fk_accounting_sync_records_released_by" FOREIGN KEY ("released_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_accounting_sync_records_skipped_by" FOREIGN KEY ("skipped_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_sync_records_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication')),
    CONSTRAINT "ck_accounting_sync_records_operation" CHECK ("operation" IN ('Create', 'Update', 'Void')),
    CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'DependencyOf', 'SafetyNet', 'Backfill')),
    CONSTRAINT "ck_accounting_sync_records_status" CHECK ("status" IN ('Queued', 'AwaitingApproval', 'InFlight', 'Retrying', 'Synced', 'Blocked', 'DeadLettered', 'Skipped', 'Superseded')),
    CONSTRAINT "ck_accounting_sync_records_error_category" CHECK ("error_category" IS NULL OR "error_category" IN ('Transient', 'RateLimited', 'Auth', 'Validation', 'Mapping', 'ClosedPeriod', 'Currency', 'Duplicate', 'NotFound', 'Conflict', 'Configuration')),
    CONSTRAINT "ck_accounting_sync_records_revision" CHECK ("revision" >= 1),
    CONSTRAINT "ck_accounting_sync_records_attempts" CHECK ("attempt_count" >= 0),
    CONSTRAINT "ck_accounting_sync_records_synced" CHECK ("status" <> 'Synced' OR ("synced_at" IS NOT NULL)),
    CONSTRAINT "ck_accounting_sync_records_skipped" CHECK ("status" <> 'Skipped' OR "skipped_reason" IS NOT NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_sync_records_idempotency" ON "accounting_sync_records"("organization_id", "business_unit_id", "connection_id", "idempotency_key");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_due" ON "accounting_sync_records"("organization_id", "business_unit_id", "connection_id", "status", "next_attempt_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_dispatch" ON "accounting_sync_records" ("next_attempt_at", "lease_expires_at")
    WHERE "status" IN ('Queued', 'Retrying', 'InFlight');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_object" ON "accounting_sync_records"("organization_id", "business_unit_id", "object_type", "object_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_queued" ON "accounting_sync_records"("organization_id", "business_unit_id", "connection_id", "queued_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_mappings" ON "accounting_sync_records" USING GIN ("mapping_ids");

--bun:split
-- One row per attempt to send a record, kept for the record's history.
CREATE TABLE IF NOT EXISTS "accounting_sync_attempts"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "sync_record_id" varchar(100) NOT NULL,
    "attempt_number" integer NOT NULL,
    "outcome" varchar(20) NOT NULL,
    "error_category" varchar(30),
    "error_code" varchar(50),
    "error_message" text,
    "started_at" bigint NOT NULL,
    "finished_at" bigint NOT NULL,
    "duration_ms" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_sync_attempts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_sync_attempts_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_attempts_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_attempts_record" FOREIGN KEY ("sync_record_id", "business_unit_id", "organization_id") REFERENCES "accounting_sync_records"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_accounting_sync_attempts_outcome" CHECK ("outcome" IN ('Synced', 'Retrying', 'Blocked', 'DeadLettered', 'Waiting')),
    CONSTRAINT "ck_accounting_sync_attempts_error_category" CHECK ("error_category" IS NULL OR "error_category" IN ('Transient', 'RateLimited', 'Auth', 'Validation', 'Mapping', 'ClosedPeriod', 'Currency', 'Duplicate', 'NotFound', 'Conflict', 'Configuration'))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_sync_attempts_record" ON "accounting_sync_attempts"("organization_id", "business_unit_id", "sync_record_id", "attempt_number");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_sync_attempts_created" ON "accounting_sync_attempts"("created_at");

--bun:split
-- A request to send documents dated from the start date up to when syncing
-- began, which never passed a live enqueue point. Resumable by its cursor.
CREATE TABLE IF NOT EXISTS "accounting_backfills"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "connection_id" varchar(100) NOT NULL,
    "range_start" bigint NOT NULL,
    "range_end" bigint NOT NULL,
    "object_types" text[] NOT NULL,
    "cursor" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "status" varchar(20) NOT NULL DEFAULT 'Queued',
    "enqueued_count" integer NOT NULL DEFAULT 0,
    "already_queued_count" integer NOT NULL DEFAULT 0,
    "requested_by_id" varchar(100),
    "started_at" bigint,
    "completed_at" bigint,
    "last_error" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_backfills" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_backfills_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_backfills_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_backfills_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_backfills_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_backfills_status" CHECK ("status" IN ('Queued', 'Running', 'Paused', 'Completed', 'Failed', 'Cancelled')),
    CONSTRAINT "ck_accounting_backfills_range" CHECK ("range_start" <= "range_end")
);

--bun:split
-- At most one backfill per connection is in progress at a time.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_backfills_active" ON "accounting_backfills"("organization_id", "business_unit_id", "connection_id") WHERE "status" IN ('Queued', 'Running', 'Paused');
