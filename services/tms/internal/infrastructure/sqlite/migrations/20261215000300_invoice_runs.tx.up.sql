-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000300_invoice_runs.tx.up.sql

CREATE TABLE IF NOT EXISTS "invoice_runs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "number" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Building',
    "source" TEXT NOT NULL,
    "cycle" TEXT,
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "invoice_date" INTEGER NOT NULL,
    "customer_ids" TEXT,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "group_count" INTEGER NOT NULL DEFAULT 0,
    "item_count" INTEGER NOT NULL DEFAULT 0,
    "excluded_count" INTEGER NOT NULL DEFAULT 0,
    "invoice_count" INTEGER NOT NULL DEFAULT 0,
    "total_amount" REAL NOT NULL DEFAULT 0,
    "total_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "failure_reason" TEXT,
    "built_by_id" TEXT,
    "built_at" INTEGER,
    "committed_by_id" TEXT,
    "committed_at" INTEGER,
    "canceled_by_id" TEXT,
    "canceled_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_invoice_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uk_invoice_runs_number" UNIQUE ("organization_id", "business_unit_id", "number")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_runs_scheduled_period" ON "invoice_runs" ("organization_id", "business_unit_id", "cycle", "period_end")WHERE
    "source" = 'Scheduled' AND "status" <> 'Canceled';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_runs_tenant_status" ON "invoice_runs" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "invoice_run_groups"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "run_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "group_key" TEXT NOT NULL,
    "group_label" TEXT NOT NULL,
    "split_by" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "item_count" INTEGER NOT NULL DEFAULT 0,
    "subtotal_amount" REAL NOT NULL DEFAULT 0,
    "subtotal_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "total_amount" REAL NOT NULL DEFAULT 0,
    "total_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "invoice_id" TEXT,
    "skip_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_invoice_run_groups" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_run_groups_run" FOREIGN KEY ("run_id", "organization_id", "business_unit_id") REFERENCES "invoice_runs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_groups_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_groups_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_run_groups_key" ON "invoice_run_groups" ("run_id", "organization_id", "business_unit_id", "customer_id", "group_key");

--bun:split

CREATE TABLE IF NOT EXISTS "invoice_run_group_items"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "run_id" TEXT NOT NULL,
    "group_id" TEXT NOT NULL,
    "billing_queue_item_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "order_id" TEXT,
    "pro_number" TEXT,
    "bol" TEXT,
    "po_number" TEXT,
    "service_date" INTEGER,
    "sort_key" INTEGER NOT NULL DEFAULT 0,
    "amount" REAL NOT NULL DEFAULT 0,
    "amount_minor" INTEGER NOT NULL DEFAULT 0,
    "excluded" INTEGER NOT NULL DEFAULT 0,
    "exclusion_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_invoice_run_group_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_run_group_items_run" FOREIGN KEY ("run_id", "organization_id", "business_unit_id") REFERENCES "invoice_runs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_group_items_group" FOREIGN KEY ("group_id", "organization_id", "business_unit_id") REFERENCES "invoice_run_groups"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_run_group_items_queue_item" FOREIGN KEY ("billing_queue_item_id", "organization_id", "business_unit_id") REFERENCES "billing_queue_items"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_run_group_items_queue_item" ON "invoice_run_group_items" ("run_id", "organization_id", "business_unit_id", "billing_queue_item_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_run_group_items_group" ON "invoice_run_group_items" ("group_id", "organization_id", "business_unit_id");

--bun:split

ALTER TABLE "invoices" ADD COLUMN "invoice_run_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_run" ON "invoices" ("invoice_run_id", "organization_id", "business_unit_id")WHERE
    "invoice_run_id" IS NOT NULL;
