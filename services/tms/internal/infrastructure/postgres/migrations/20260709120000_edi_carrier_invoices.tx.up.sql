CREATE TYPE "edi_carrier_invoice_reconciliation_status_enum" AS ENUM(
    'Unmatched',
    'MappingRequired',
    'Matched',
    'Variance'
);

--bun:split
CREATE TABLE IF NOT EXISTS "edi_carrier_invoices"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "edi_partner_id" varchar(100) NOT NULL,
    "inbound_message_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100),
    "tender_recipient_id" varchar(100),
    "customer_id" varchar(100),
    "invoice_number" varchar(100) NOT NULL,
    "invoice_date" bigint,
    "delivery_date" bigint,
    "shipment_reference" varchar(100),
    "bol" varchar(100),
    "pro_number" varchar(100),
    "bill_to_name" varchar(200),
    "bill_to_source_id" varchar(100),
    "currency_code" varchar(3),
    "total_amount" numeric(19, 4),
    "expected_amount" numeric(19, 4),
    "variance_amount" numeric(19, 4),
    "line_charges" jsonb NOT NULL DEFAULT '[]',
    "reference_numbers" jsonb NOT NULL DEFAULT '{}',
    "reconciliation_status" edi_carrier_invoice_reconciliation_status_enum NOT NULL DEFAULT 'Unmatched',
    "reconciliation_notes" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_edi_carrier_invoices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_carrier_invoices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_carrier_invoices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_carrier_invoices_partner" FOREIGN KEY ("edi_partner_id", "business_unit_id", "organization_id") REFERENCES "edi_partners"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_status" ON "edi_carrier_invoices"("organization_id", "business_unit_id", "reconciliation_status", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_partner_invoice_number" ON "edi_carrier_invoices"("organization_id", "business_unit_id", "edi_partner_id", "invoice_number");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_shipment" ON "edi_carrier_invoices"("shipment_id")
WHERE
    "shipment_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_inbound_message" ON "edi_carrier_invoices"("inbound_message_id");
