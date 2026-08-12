-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260720000000_order_charges.tx.up.sql

CREATE TABLE IF NOT EXISTS "order_charges"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "order_id" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "amount" REAL NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_order_charges" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_order_charges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_order_charges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_order_charges_order" FOREIGN KEY ("order_id", "organization_id", "business_unit_id") REFERENCES "orders"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_order_charges_order ON order_charges ("order_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_order_charges_business_unit ON order_charges ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_order_charges_organization ON order_charges ("organization_id");
