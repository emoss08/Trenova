-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260411183000_add_bank_receipt_work_items.tx.up.sql

CREATE TABLE IF NOT EXISTS bank_receipt_work_items(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "bank_receipt_id" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "assigned_to_user_id" TEXT,
    "assigned_at" INTEGER,
    "resolution_type" TEXT,
    "resolution_note" TEXT,
    "resolved_by_user_id" TEXT,
    "resolved_at" INTEGER,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_bank_receipt_work_items PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_bank_receipt_work_items_receipt FOREIGN KEY (bank_receipt_id, organization_id, business_unit_id) REFERENCES bank_receipts(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_bank_receipt_work_items_assigned_to FOREIGN KEY (assigned_to_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipt_work_items_resolved_by FOREIGN KEY (resolved_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipt_work_items_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_bank_receipt_work_items_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_bank_receipt_work_items_active
    ON bank_receipt_work_items (organization_id, business_unit_id, bank_receipt_id)WHERE status IN ('Open', 'Assigned', 'InReview');
