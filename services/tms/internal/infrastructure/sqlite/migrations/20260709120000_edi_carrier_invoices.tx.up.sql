-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260709120000_edi_carrier_invoices.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_carrier_invoices"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "edi_partner_id" TEXT NOT NULL,
    "inbound_message_id" TEXT NOT NULL,
    "shipment_id" TEXT,
    "tender_recipient_id" TEXT,
    "customer_id" TEXT,
    "invoice_number" TEXT NOT NULL,
    "invoice_date" INTEGER,
    "delivery_date" INTEGER,
    "shipment_reference" TEXT,
    "bol" TEXT,
    "pro_number" TEXT,
    "bill_to_name" TEXT,
    "bill_to_source_id" TEXT,
    "currency_code" TEXT,
    "total_amount" REAL,
    "expected_amount" REAL,
    "variance_amount" REAL,
    "line_charges" TEXT NOT NULL DEFAULT '[]',
    "reference_numbers" TEXT NOT NULL DEFAULT '{}',
    "reconciliation_status" TEXT NOT NULL DEFAULT 'Unmatched',
    "reconciliation_notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_carrier_invoices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_carrier_invoices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_carrier_invoices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_carrier_invoices_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_status" ON "edi_carrier_invoices" ("organization_id", "business_unit_id", "reconciliation_status", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_partner_invoice_number" ON "edi_carrier_invoices" ("organization_id", "business_unit_id", "edi_partner_id", "invoice_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_shipment" ON "edi_carrier_invoices" ("shipment_id")WHERE
    "shipment_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_inbound_message" ON "edi_carrier_invoices" ("inbound_message_id");
