-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260411165000_add_customer_ledger_entries.tx.up.sql

CREATE TABLE IF NOT EXISTS customer_ledger_entries(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "source_object_type" TEXT NOT NULL,
    "source_object_id" TEXT NOT NULL,
    "source_event_type" TEXT NOT NULL,
    "related_invoice_id" TEXT,
    "document_number" TEXT,
    "transaction_date" INTEGER NOT NULL,
    "line_number" INTEGER NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "created_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_customer_ledger_entries PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_customer_ledger_entries_source_line UNIQUE (organization_id, business_unit_id, source_event_type, source_object_id, line_number),
    CONSTRAINT fk_customer_ledger_entries_customer FOREIGN KEY (customer_id, organization_id, business_unit_id) REFERENCES customers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_customer_ledger_entries_invoice FOREIGN KEY (related_invoice_id, organization_id, business_unit_id) REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_customer_ledger_entries_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_customer_ledger_entries_customer_date ON customer_ledger_entries (organization_id, business_unit_id, customer_id, transaction_date);
