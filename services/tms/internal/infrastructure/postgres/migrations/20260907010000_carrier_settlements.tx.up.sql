ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'CarrierSettlementPosted';

--bun:split
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'CarrierSettlementVoided';

--bun:split
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'CarrierSettlementPaid';

--bun:split
ALTER TABLE edi_carrier_invoices
    ADD COLUMN IF NOT EXISTS carrier_id VARCHAR(100);

--bun:split
ALTER TABLE edi_carrier_invoices
    ADD CONSTRAINT fk_edi_carrier_invoices_carrier FOREIGN KEY (carrier_id, organization_id, business_unit_id) REFERENCES carriers(id, organization_id, business_unit_id) ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_edi_carrier_invoices_carrier ON edi_carrier_invoices(organization_id, business_unit_id, carrier_id) WHERE carrier_id IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS carrier_settlement_controls(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    pay_trigger VARCHAR(50) NOT NULL DEFAULT 'ShipmentDelivered',
    pay_period_frequency VARCHAR(50) NOT NULL DEFAULT 'Weekly',
    period_end_day_of_week INTEGER NOT NULL DEFAULT 6,
    pay_delay_days INTEGER NOT NULL DEFAULT 5,
    auto_generate_batches BOOLEAN NOT NULL DEFAULT FALSE,
    auto_post_on_approve BOOLEAN NOT NULL DEFAULT FALSE,
    variance_tolerance_minor BIGINT NOT NULL DEFAULT 0,
    default_ap_account_id VARCHAR(100),
    default_purchased_transportation_account_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_carrier_settlement_controls PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_carrier_settlement_controls_period_end_day CHECK (period_end_day_of_week BETWEEN 0 AND 6),
    CONSTRAINT chk_carrier_settlement_controls_pay_delay CHECK (pay_delay_days BETWEEN 0 AND 30),
    CONSTRAINT chk_carrier_settlement_controls_tolerance CHECK (variance_tolerance_minor >= 0),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE,
    FOREIGN KEY (default_ap_account_id, organization_id, business_unit_id) REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (default_purchased_transportation_account_id, organization_id, business_unit_id) REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_settlement_controls_org ON carrier_settlement_controls(organization_id, business_unit_id);

--bun:split
CREATE TABLE IF NOT EXISTS carrier_settlement_batches(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Open',
    name VARCHAR(100) NOT NULL,
    period_start BIGINT NOT NULL,
    period_end BIGINT NOT NULL,
    pay_date BIGINT NOT NULL,
    settlement_count INTEGER NOT NULL DEFAULT 0,
    total_gross_minor BIGINT NOT NULL DEFAULT 0,
    total_net_minor BIGINT NOT NULL DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    notes TEXT,
    generated_by_id VARCHAR(100),
    generated_at BIGINT,
    completed_at BIGINT,
    canceled_by_id VARCHAR(100),
    canceled_at BIGINT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_carrier_settlement_batches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_carrier_settlement_batches_period CHECK (period_end > period_start),
    FOREIGN KEY (generated_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (canceled_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_settlement_batches_status ON carrier_settlement_batches(organization_id, business_unit_id, status, period_end DESC);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_settlement_batches_period ON carrier_settlement_batches(organization_id, business_unit_id, period_start, period_end) WHERE status = 'Open';

--bun:split
CREATE TABLE IF NOT EXISTS carrier_settlements(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    carrier_id VARCHAR(100) NOT NULL,
    batch_id VARCHAR(100),
    settlement_number VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Draft',
    period_start BIGINT NOT NULL,
    period_end BIGINT NOT NULL,
    pay_date BIGINT NOT NULL,
    gross_cost_minor BIGINT NOT NULL DEFAULT 0,
    adjustments_minor BIGINT NOT NULL DEFAULT 0,
    net_payable_minor BIGINT NOT NULL DEFAULT 0,
    shipment_count INTEGER NOT NULL DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    notes TEXT,
    submitted_by_id VARCHAR(100),
    submitted_at BIGINT,
    approved_by_id VARCHAR(100),
    approved_at BIGINT,
    posted_by_id VARCHAR(100),
    posted_at BIGINT,
    posted_journal_batch_id VARCHAR(100),
    paid_at BIGINT,
    paid_by_id VARCHAR(100),
    payment_method VARCHAR(50),
    payment_reference VARCHAR(100),
    paid_journal_batch_id VARCHAR(100),
    voided_by_id VARCHAR(100),
    voided_at BIGINT,
    void_reason TEXT,
    void_journal_batch_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_carrier_settlements PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_carrier_settlements_period CHECK (period_end > period_start),
    FOREIGN KEY (carrier_id, organization_id, business_unit_id) REFERENCES carriers(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (batch_id, organization_id, business_unit_id) REFERENCES carrier_settlement_batches(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (submitted_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (approved_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (posted_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (paid_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (voided_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_settlements_number ON carrier_settlements(organization_id, business_unit_id, settlement_number);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_settlements_carrier ON carrier_settlements(organization_id, business_unit_id, carrier_id, period_end DESC);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_settlements_status ON carrier_settlements(organization_id, business_unit_id, status);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_settlements_batch ON carrier_settlements(organization_id, business_unit_id, batch_id) WHERE batch_id IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS carrier_settlement_lines(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    settlement_id VARCHAR(100) NOT NULL,
    line_number INTEGER NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    description VARCHAR(255) NOT NULL,
    amount_minor BIGINT NOT NULL,
    cost_event_id VARCHAR(100),
    gl_account_id VARCHAR(100),
    shipment_id VARCHAR(100),
    move_id VARCHAR(100),
    pro_number VARCHAR(100),
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_carrier_settlement_lines PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_carrier_settlement_lines_number UNIQUE (settlement_id, organization_id, business_unit_id, line_number),
    FOREIGN KEY (settlement_id, organization_id, business_unit_id) REFERENCES carrier_settlements(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (gl_account_id, organization_id, business_unit_id) REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_settlement_lines_settlement ON carrier_settlement_lines(organization_id, business_unit_id, settlement_id);

--bun:split
CREATE TABLE IF NOT EXISTS carrier_cost_events(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    carrier_id VARCHAR(100) NOT NULL,
    carrier_assignment_id VARCHAR(100),
    shipment_id VARCHAR(100),
    move_id VARCHAR(100),
    settlement_id VARCHAR(100),
    event_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending',
    idempotency_key VARCHAR(255) NOT NULL,
    event_date BIGINT NOT NULL,
    amount_minor BIGINT NOT NULL DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    description VARCHAR(255),
    pro_number VARCHAR(100),
    assignment_version BIGINT NOT NULL DEFAULT 0,
    voided_at BIGINT,
    void_reason TEXT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_carrier_cost_events PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_carrier_cost_events_amount CHECK (event_type = 'Adjustment' OR amount_minor >= 0),
    FOREIGN KEY (carrier_id, organization_id, business_unit_id) REFERENCES carriers(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (shipment_id, organization_id, business_unit_id) REFERENCES shipments(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_cost_events_idempotency ON carrier_cost_events(organization_id, business_unit_id, idempotency_key);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_cost_events_carrier ON carrier_cost_events(organization_id, business_unit_id, carrier_id, status, event_date DESC);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_cost_events_shipment ON carrier_cost_events(organization_id, business_unit_id, shipment_id) WHERE shipment_id IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_cost_events_settlement ON carrier_cost_events(organization_id, business_unit_id, settlement_id) WHERE settlement_id IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS carrier_ledger_entries(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    carrier_id VARCHAR(100) NOT NULL,
    entry_type VARCHAR(50) NOT NULL,
    source_object_type VARCHAR(50) NOT NULL,
    source_object_id VARCHAR(100) NOT NULL,
    source_event_type VARCHAR(100) NOT NULL,
    related_settlement_id VARCHAR(100),
    journal_batch_id VARCHAR(100),
    document_number VARCHAR(100),
    transaction_date BIGINT NOT NULL,
    line_number INTEGER NOT NULL,
    amount_minor BIGINT NOT NULL,
    created_by_id VARCHAR(100) NOT NULL,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_carrier_ledger_entries PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_carrier_ledger_entries_source_line UNIQUE (organization_id, business_unit_id, source_event_type, source_object_id, line_number),
    CONSTRAINT fk_carrier_ledger_entries_carrier FOREIGN KEY (carrier_id, organization_id, business_unit_id) REFERENCES carriers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_carrier_ledger_entries_settlement FOREIGN KEY (related_settlement_id, organization_id, business_unit_id) REFERENCES carrier_settlements(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_carrier_ledger_entries_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_ledger_entries_carrier_date ON carrier_ledger_entries(organization_id, business_unit_id, carrier_id, transaction_date);

--bun:split
CREATE TABLE IF NOT EXISTS carrier_invoice_matches(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    edi_carrier_invoice_id VARCHAR(100),
    document_ai_extraction_id VARCHAR(100),
    carrier_id VARCHAR(100) NOT NULL,
    carrier_assignment_id VARCHAR(100) NOT NULL,
    carrier_settlement_id VARCHAR(100),
    adjustment_cost_event_id VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'Suggested',
    invoice_number VARCHAR(100),
    invoice_total_minor BIGINT NOT NULL DEFAULT 0,
    expected_total_minor BIGINT NOT NULL DEFAULT 0,
    variance_minor BIGINT NOT NULL DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    resolution_note TEXT,
    resolved_by_id VARCHAR(100),
    resolved_at BIGINT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_carrier_invoice_matches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_carrier_invoice_matches_source CHECK ((edi_carrier_invoice_id IS NULL) <> (document_ai_extraction_id IS NULL)),
    FOREIGN KEY (carrier_id, organization_id, business_unit_id) REFERENCES carriers(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (carrier_assignment_id, organization_id, business_unit_id) REFERENCES carrier_assignments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (carrier_settlement_id, organization_id, business_unit_id) REFERENCES carrier_settlements(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (resolved_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_invoice_matches_status ON carrier_invoice_matches(organization_id, business_unit_id, status);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_invoice_matches_carrier ON carrier_invoice_matches(organization_id, business_unit_id, carrier_id);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_invoice_matches_edi_invoice ON carrier_invoice_matches(organization_id, business_unit_id, edi_carrier_invoice_id) WHERE edi_carrier_invoice_id IS NOT NULL AND status <> 'Rejected';
