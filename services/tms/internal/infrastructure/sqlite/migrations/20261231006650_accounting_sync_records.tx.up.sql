-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006650_accounting_sync_records.tx.up.sql

ALTER TABLE "accounting_connections" ADD COLUMN "sync_start_date" INTEGER;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "sync_enabled_at" INTEGER;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "auto_sync" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "paused_at" INTEGER;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "paused_by_id" TEXT;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "paused_reason" TEXT;

--bun:split

UPDATE "accounting_connections" SET "setup_step" = 'StartDate' WHERE "setup_step" = 'Complete' AND "sync_start_date" IS NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "accounting_sync_records"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "connection_id" TEXT NOT NULL,
    "object_type" TEXT NOT NULL,
    "object_id" TEXT NOT NULL,
    "object_number" TEXT,
    "operation" TEXT NOT NULL,
    "source_event" TEXT NOT NULL,
    "idempotency_key" TEXT NOT NULL,
    "request_id" TEXT NOT NULL,
    "revision" INTEGER NOT NULL DEFAULT 1,
    "document_date" INTEGER,
    "depends_on_record_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Queued',
    "attempt_count" INTEGER NOT NULL DEFAULT 0,
    "next_attempt_at" INTEGER,
    "lease_expires_at" INTEGER,
    "external_id" TEXT,
    "external_doc_number" TEXT,
    "external_url" TEXT,
    "external_refs" TEXT NOT NULL DEFAULT '{}',
    "payload_hash" TEXT,
    "payload" TEXT,
    "mapping_ids" TEXT NOT NULL DEFAULT '[]',
    "error_category" TEXT,
    "error_code" TEXT,
    "error_message" TEXT,
    "resolution" TEXT,
    "queued_at" INTEGER NOT NULL,
    "started_at" INTEGER,
    "synced_at" INTEGER,
    "released_by_id" TEXT,
    "skipped_by_id" TEXT,
    "skipped_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_sync_records" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_sync_records_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_sync_records_idempotency" ON "accounting_sync_records" ("organization_id", "business_unit_id", "connection_id", "idempotency_key");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_due" ON "accounting_sync_records" ("organization_id", "business_unit_id", "connection_id", "status", "next_attempt_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_dispatch" ON "accounting_sync_records" ("next_attempt_at", "lease_expires_at")WHERE "status" IN ('Queued', 'Retrying', 'InFlight');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_object" ON "accounting_sync_records" ("organization_id", "business_unit_id", "object_type", "object_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_queued" ON "accounting_sync_records" ("organization_id", "business_unit_id", "connection_id", "queued_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "accounting_sync_attempts"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "sync_record_id" TEXT NOT NULL,
    "attempt_number" INTEGER NOT NULL,
    "outcome" TEXT NOT NULL,
    "error_category" TEXT,
    "error_code" TEXT,
    "error_message" TEXT,
    "started_at" INTEGER NOT NULL,
    "finished_at" INTEGER NOT NULL,
    "duration_ms" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_sync_attempts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_sync_attempts_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_attempts_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_attempts_record" FOREIGN KEY ("sync_record_id", "business_unit_id", "organization_id") REFERENCES "accounting_sync_records"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_accounting_sync_attempts_outcome" CHECK ("outcome" IN ('Synced', 'Retrying', 'Blocked', 'DeadLettered', 'Waiting')),
    CONSTRAINT "ck_accounting_sync_attempts_error_category" CHECK ("error_category" IS NULL OR "error_category" IN ('Transient', 'RateLimited', 'Auth', 'Validation', 'Mapping', 'ClosedPeriod', 'Currency', 'Duplicate', 'NotFound', 'Conflict', 'Configuration'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_attempts_record" ON "accounting_sync_attempts" ("organization_id", "business_unit_id", "sync_record_id", "attempt_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_attempts_created" ON "accounting_sync_attempts" ("created_at");

--bun:split

CREATE TABLE IF NOT EXISTS "accounting_backfills"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "connection_id" TEXT NOT NULL,
    "range_start" INTEGER NOT NULL,
    "range_end" INTEGER NOT NULL,
    "object_types" TEXT NOT NULL,
    "cursor" TEXT NOT NULL DEFAULT '{}',
    "status" TEXT NOT NULL DEFAULT 'Queued',
    "enqueued_count" INTEGER NOT NULL DEFAULT 0,
    "already_queued_count" INTEGER NOT NULL DEFAULT 0,
    "requested_by_id" TEXT,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "last_error" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_backfills" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_backfills_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_backfills_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_backfills_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_backfills_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_backfills_status" CHECK ("status" IN ('Queued', 'Running', 'Paused', 'Completed', 'Failed', 'Cancelled')),
    CONSTRAINT "ck_accounting_backfills_range" CHECK ("range_start" <= "range_end")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_backfills_active" ON "accounting_backfills" ("organization_id", "business_unit_id", "connection_id")WHERE "status" IN ('Queued', 'Running', 'Paused');
