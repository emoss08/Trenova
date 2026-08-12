-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260408120000_add_invoices.tx.up.sql

CREATE TABLE IF NOT EXISTS "invoices"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "billing_queue_item_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "number" TEXT NOT NULL,
    "bill_type" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "payment_term" TEXT NOT NULL,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "invoice_date" INTEGER NOT NULL,
    "due_date" INTEGER,
    "posted_at" INTEGER,
    "shipment_pro_number" TEXT,
    "shipment_bol" TEXT,
    "service_date" INTEGER,
    "bill_to_name" TEXT NOT NULL,
    "bill_to_code" TEXT,
    "bill_to_address_line_1" TEXT,
    "bill_to_address_line_2" TEXT,
    "bill_to_city" TEXT,
    "bill_to_state" TEXT,
    "bill_to_postal_code" TEXT,
    "bill_to_country" TEXT,
    "subtotal_amount" REAL NOT NULL DEFAULT 0,
    "other_amount" REAL NOT NULL DEFAULT 0,
    "total_amount" REAL NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_invoices" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_invoices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_invoices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_invoices_billing_queue_item" FOREIGN KEY ("billing_queue_item_id", "organization_id", "business_unit_id") REFERENCES "billing_queue_items"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_invoices_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_invoices_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "uk_invoices_billing_queue_item" UNIQUE ("billing_queue_item_id", "organization_id", "business_unit_id"),
    CONSTRAINT "uk_invoices_number" UNIQUE ("organization_id", "business_unit_id", "number")
);

--bun:split

CREATE TABLE IF NOT EXISTS "invoice_lines"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "line_number" INTEGER NOT NULL,
    "type" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "quantity" REAL NOT NULL DEFAULT 0,
    "unit_price" REAL NOT NULL DEFAULT 0,
    "amount" REAL NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_invoice_lines" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_invoice_lines_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "uk_invoice_lines_number" UNIQUE ("invoice_id", "organization_id", "business_unit_id", "line_number")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_tenant_status" ON "invoices" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoices_shipment" ON "invoices" ("organization_id", "business_unit_id", "shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_lines_invoice" ON "invoice_lines" ("invoice_id", "organization_id", "business_unit_id");
