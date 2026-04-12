CREATE TABLE IF NOT EXISTS customer_ledger_entries(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    customer_id VARCHAR(100) NOT NULL,
    source_object_type VARCHAR(50) NOT NULL,
    source_object_id VARCHAR(100) NOT NULL,
    source_event_type VARCHAR(100) NOT NULL,
    related_invoice_id VARCHAR(100),
    document_number VARCHAR(100),
    transaction_date BIGINT NOT NULL,
    line_number INTEGER NOT NULL,
    amount_minor BIGINT NOT NULL,
    created_by_id VARCHAR(100) NOT NULL,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_customer_ledger_entries PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_customer_ledger_entries_source_line UNIQUE (organization_id, business_unit_id, source_event_type, source_object_id, line_number),
    CONSTRAINT fk_customer_ledger_entries_customer FOREIGN KEY (customer_id, organization_id, business_unit_id) REFERENCES customers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_customer_ledger_entries_invoice FOREIGN KEY (related_invoice_id, organization_id, business_unit_id) REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_customer_ledger_entries_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_customer_ledger_entries_customer_date ON customer_ledger_entries(organization_id, business_unit_id, customer_id, transaction_date);
