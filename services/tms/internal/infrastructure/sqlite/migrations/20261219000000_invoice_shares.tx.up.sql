-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261219000000_invoice_shares.tx.up.sql

CREATE TABLE IF NOT EXISTS "invoice_shares"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "shared_with_id" TEXT NOT NULL,
    "shared_by_id" TEXT NOT NULL,
    "note" TEXT,
    "tab" TEXT NOT NULL DEFAULT 'overview',
    "share_count" INTEGER NOT NULL DEFAULT 1,
    "first_shared_at" INTEGER NOT NULL,
    "last_shared_at" INTEGER NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_invoice_shares" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_invoice_shares_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_shares_shared_with" FOREIGN KEY ("shared_with_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_invoice_shares_shared_by" FOREIGN KEY ("shared_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_invoice_shares_tab" CHECK ("tab" IN ('overview', 'delivery', 'charges', 'documents', 'activity')),
    CONSTRAINT "chk_invoice_shares_share_count" CHECK ("share_count" >= 1)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoice_shares_recipient" ON "invoice_shares" ("invoice_id", "organization_id", "business_unit_id", "shared_with_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_invoice_shares_shared_with" ON "invoice_shares" ("shared_with_id", "organization_id", "business_unit_id");
