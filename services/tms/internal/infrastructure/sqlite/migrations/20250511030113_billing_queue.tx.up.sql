-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250511030113_billing_queue.tx.up.sql

CREATE TABLE IF NOT EXISTS billing_queue_items(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "assigned_biller_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'ReadyForReview',
    "bill_type" TEXT NOT NULL DEFAULT 'Invoice',
    "review_notes" TEXT,
    "exception_notes" TEXT,
    "review_started_at" INTEGER,
    "review_completed_at" INTEGER,
    "canceled_by_id" TEXT,
    "canceled_at" INTEGER,
    "cancel_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_billing_queue_items" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_billing_queue_items_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_items_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_items_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_billing_queue_items_assigned_biller" FOREIGN KEY ("assigned_biller_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_billing_queue_items_canceled_by" FOREIGN KEY ("canceled_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "ck_billing_queue_items_status" CHECK ("status" IN ('ReadyForReview', 'InReview', 'Approved', 'Canceled', 'Exception')),
    CONSTRAINT "ck_billing_queue_items_exception_notes" CHECK ("status" != 'Exception' OR "exception_notes" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_billing_queue_items_shipment_bill_type
ON billing_queue_items ("shipment_id", "organization_id", "business_unit_id", "bill_type");

--bun:split

CREATE INDEX IF NOT EXISTS idx_billing_queue_items_shipment_id ON billing_queue_items ("shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_billing_queue_items_organization_id ON billing_queue_items ("organization_id");
