-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260411193000_add_bank_receipt_import_batches.tx.up.sql

CREATE TABLE IF NOT EXISTS bank_receipt_import_batches(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "reference" TEXT,
    "status" TEXT NOT NULL,
    "imported_count" INTEGER NOT NULL DEFAULT 0,
    "matched_count" INTEGER NOT NULL DEFAULT 0,
    "exception_count" INTEGER NOT NULL DEFAULT 0,
    "imported_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "matched_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "exception_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_bank_receipt_import_batches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_bank_receipt_import_batches_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_bank_receipt_import_batches_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

ALTER TABLE "bank_receipts" ADD COLUMN "import_batch_id" TEXT;
