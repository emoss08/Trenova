--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- A run is the persisted artifact behind statement billing: the proposal an
-- operator reviews before it becomes invoices, and the thing a scheduled sweep
-- resumes from when it fails halfway.
CREATE TYPE "invoice_run_status_enum" AS ENUM(
    'Building',
    'Ready',
    'Committing',
    'Committed',
    'Failed',
    'Canceled'
);

--bun:split
CREATE TYPE "invoice_run_source_enum" AS ENUM(
    'Manual',
    'Scheduled'
);

--bun:split
CREATE TYPE "invoice_run_group_status_enum" AS ENUM(
    'Pending',
    'Committed',
    'Skipped',
    'Failed'
);

--bun:split
CREATE TABLE IF NOT EXISTS "invoice_runs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "number" varchar(100) NOT NULL,
    "status" invoice_run_status_enum NOT NULL DEFAULT 'Building',
    "source" invoice_run_source_enum NOT NULL,
    "cycle" customer_billing_cycle_enum,
    "period_start" bigint NOT NULL,
    "period_end" bigint NOT NULL,
    "invoice_date" bigint NOT NULL,
    "customer_ids" text[],
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "group_count" integer NOT NULL DEFAULT 0,
    "item_count" integer NOT NULL DEFAULT 0,
    "excluded_count" integer NOT NULL DEFAULT 0,
    "invoice_count" integer NOT NULL DEFAULT 0,
    "total_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "total_amount_minor" bigint NOT NULL DEFAULT 0,
    "failure_reason" text,
    "built_by_id" varchar(100),
    "built_at" bigint,
    "committed_by_id" varchar(100),
    "committed_at" bigint,
    "canceled_by_id" varchar(100),
    "canceled_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_invoice_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uk_invoice_runs_number" UNIQUE ("organization_id", "business_unit_id", "number")
);

--bun:split
-- One scheduled run per cadence tick. Two workers racing the same cron must not
-- both build the same period; a canceled run may be rebuilt.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_runs_scheduled_period" ON "invoice_runs"("organization_id", "business_unit_id", "cycle", "period_end")
WHERE
    "source" = 'Scheduled' AND "status" <> 'Canceled';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_runs_tenant_status" ON "invoice_runs"("organization_id", "business_unit_id", "status");

--bun:split
CREATE TABLE IF NOT EXISTS "invoice_run_groups"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "group_key" varchar(255) NOT NULL,
    "group_label" varchar(255) NOT NULL,
    "split_by" invoice_split_key_enum NOT NULL,
    "status" invoice_run_group_status_enum NOT NULL DEFAULT 'Pending',
    "item_count" integer NOT NULL DEFAULT 0,
    "subtotal_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "subtotal_amount_minor" bigint NOT NULL DEFAULT 0,
    "total_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "total_amount_minor" bigint NOT NULL DEFAULT 0,
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "invoice_id" varchar(100),
    "skip_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_invoice_run_groups" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_run_groups_run" FOREIGN KEY ("run_id", "organization_id", "business_unit_id") REFERENCES "invoice_runs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_groups_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_groups_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_run_groups_key" ON "invoice_run_groups"("run_id", "organization_id", "business_unit_id", "customer_id", "group_key");

--bun:split
CREATE TABLE IF NOT EXISTS "invoice_run_group_items"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "group_id" varchar(100) NOT NULL,
    "billing_queue_item_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "order_id" varchar(100),
    "pro_number" varchar(100),
    "bol" varchar(100),
    "po_number" varchar(100),
    "service_date" bigint,
    "sort_key" integer NOT NULL DEFAULT 0,
    "amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "amount_minor" bigint NOT NULL DEFAULT 0,
    "excluded" boolean NOT NULL DEFAULT FALSE,
    "exclusion_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_invoice_run_group_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_run_group_items_run" FOREIGN KEY ("run_id", "organization_id", "business_unit_id") REFERENCES "invoice_runs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_group_items_group" FOREIGN KEY ("group_id", "organization_id", "business_unit_id") REFERENCES "invoice_run_groups"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_group_items_queue_item" FOREIGN KEY ("billing_queue_item_id", "organization_id", "business_unit_id") REFERENCES "billing_queue_items"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- A queue item appears at most once in a run. Across runs it is deliberately
-- unconstrained: the real double-bill guard is billing_queue_items.invoice_id,
-- re-checked under SELECT ... FOR UPDATE inside the commit transaction. An index
-- here could only express "one open run per item" by denormalising the run's
-- status onto every item, which would then have to be kept in step.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_run_group_items_queue_item" ON "invoice_run_group_items"("run_id", "organization_id", "business_unit_id", "billing_queue_item_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoice_run_group_items_group" ON "invoice_run_group_items"("group_id", "organization_id", "business_unit_id");

--bun:split
-- Deferred from the invoice-scope migration, which could not reference a table
-- that did not exist yet.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "invoice_run_id" varchar(100);

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "fk_invoices_invoice_run" FOREIGN KEY ("invoice_run_id", "organization_id", "business_unit_id") REFERENCES "invoice_runs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoices_run" ON "invoices"("invoice_run_id", "organization_id", "business_unit_id")
WHERE
    "invoice_run_id" IS NOT NULL;
