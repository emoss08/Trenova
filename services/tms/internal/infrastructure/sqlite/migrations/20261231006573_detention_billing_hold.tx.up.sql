-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006573_detention_billing_hold.tx.up.sql

ALTER TABLE "invoice_lines" ADD COLUMN "additional_charge_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_lines_additional_charge" ON "invoice_lines" ("additional_charge_id", "organization_id", "business_unit_id")WHERE
    "additional_charge_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_billing_hold" ON "detention_occurrences" ("organization_id", "business_unit_id", "shipment_id")WHERE
    "status" = 'Pending'
    AND "requires_approval";
