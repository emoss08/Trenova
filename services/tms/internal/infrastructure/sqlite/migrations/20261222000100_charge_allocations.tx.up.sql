-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261222000100_charge_allocations.tx.up.sql

CREATE TABLE IF NOT EXISTS "charge_allocations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT,
    "additional_charge_id" TEXT,
    "order_charge_id" TEXT,
    "charge_kind" TEXT NOT NULL,
    "bill_to_customer_id" TEXT NOT NULL,
    "method" TEXT NOT NULL,
    "percent" REAL,
    "amount" REAL,
    "sequence" INTEGER NOT NULL DEFAULT 0,
    "invoice_id" TEXT,
    "invoiced_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_charge_allocations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_charge_allocations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_charge_allocations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_charge_allocations_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_charge_allocations_additional_charge" FOREIGN KEY ("additional_charge_id") REFERENCES "additional_charges"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_charge_allocations_order_charge" FOREIGN KEY ("order_charge_id", "business_unit_id", "organization_id") REFERENCES "order_charges"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_charge_allocations_bill_to_customer" FOREIGN KEY ("bill_to_customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_charge_allocations_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_charge_allocations_target" CHECK (("charge_kind" = 'Freight' AND "shipment_id" IS NOT NULL AND "additional_charge_id" IS NULL AND "order_charge_id" IS NULL) OR ("charge_kind" = 'Accessorial' AND "shipment_id" IS NOT NULL AND "additional_charge_id" IS NOT NULL AND "order_charge_id" IS NULL) OR ("charge_kind" = 'OrderCharge' AND "shipment_id" IS NULL AND "additional_charge_id" IS NULL AND "order_charge_id" IS NOT NULL)),
    CONSTRAINT "chk_charge_allocations_method" CHECK (("method" = 'Percent' AND "percent" IS NOT NULL AND "percent" > 0 AND "percent" <= 100 AND "amount" IS NULL) OR ("method" = 'Amount' AND "amount" IS NOT NULL AND "amount" > 0 AND "percent" IS NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_charge_allocations_freight_payer" ON "charge_allocations" ("shipment_id", "bill_to_customer_id", "organization_id", "business_unit_id")WHERE
    "charge_kind" = 'Freight';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_charge_allocations_accessorial_payer" ON "charge_allocations" ("additional_charge_id", "bill_to_customer_id", "organization_id", "business_unit_id")WHERE
    "charge_kind" = 'Accessorial';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_charge_allocations_order_charge_payer" ON "charge_allocations" ("order_charge_id", "bill_to_customer_id", "organization_id", "business_unit_id")WHERE
    "charge_kind" = 'OrderCharge';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_charge_allocations_shipment" ON "charge_allocations" ("shipment_id", "organization_id", "business_unit_id")WHERE
    "shipment_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_charge_allocations_order_charge" ON "charge_allocations" ("order_charge_id", "organization_id", "business_unit_id")WHERE
    "order_charge_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_charge_allocations_payer" ON "charge_allocations" ("bill_to_customer_id", "organization_id", "business_unit_id");
