ALTER TABLE carrier_settlements
    ADD COLUMN IF NOT EXISTS posted_expense_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS posted_ap_account_id VARCHAR(100);

--bun:split
-- Paid and void journals must hit the accounts the posting used, so the
-- resolved accounts are stamped at post time and referenced thereafter.
ALTER TABLE carrier_settlements
    ADD CONSTRAINT fk_carrier_settlements_posted_expense_account FOREIGN KEY (posted_expense_account_id, organization_id, business_unit_id) REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_carrier_settlements_posted_ap_account FOREIGN KEY (posted_ap_account_id, organization_id, business_unit_id) REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
-- One settlement per carrier per period; a voided settlement releases the slot.
CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_settlements_carrier_period ON carrier_settlements(organization_id, business_unit_id, carrier_id, period_start, period_end)
WHERE
    status <> 'Voided';

--bun:split
-- Document-AI-sourced invoices get the same one-open-match guarantee EDI
-- invoices already have.
CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_invoice_matches_extraction ON carrier_invoice_matches(document_ai_extraction_id, organization_id, business_unit_id)
WHERE
    document_ai_extraction_id IS NOT NULL AND status <> 'Rejected';

--bun:split
ALTER TABLE carrier_invoice_matches
    ADD COLUMN IF NOT EXISTS adjustment_cost_event_id VARCHAR(100);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_invoice_matches_adjustment_event ON carrier_invoice_matches(adjustment_cost_event_id, organization_id)
WHERE
    adjustment_cost_event_id IS NOT NULL;

--bun:split
-- Posted accounting must not vanish with a shipment delete: the cost-event FK
-- to shipments was created unnamed with ON DELETE CASCADE, so it is located
-- through the catalog and recreated as RESTRICT.
DO $$
DECLARE
    c record;
BEGIN
    FOR c IN
    SELECT conname
    FROM pg_constraint
    WHERE conrelid = 'carrier_cost_events'::regclass
        AND contype = 'f'
        AND confrelid = 'shipments'::regclass LOOP
        EXECUTE format('ALTER TABLE carrier_cost_events DROP CONSTRAINT %I', c.conname);
    END LOOP;
END
$$;

--bun:split
ALTER TABLE carrier_cost_events
    ADD CONSTRAINT fk_carrier_cost_events_shipment FOREIGN KEY (shipment_id, organization_id, business_unit_id) REFERENCES shipments(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
ALTER TABLE carrier_cost_events
    ADD CONSTRAINT fk_carrier_cost_events_shipment_move FOREIGN KEY (move_id, organization_id, business_unit_id) REFERENCES shipment_moves(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_carrier_cost_events_carrier_assignment FOREIGN KEY (carrier_assignment_id, organization_id, business_unit_id) REFERENCES carrier_assignments(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
ALTER TABLE carrier_settlement_lines
    ADD CONSTRAINT fk_carrier_settlement_lines_cost_event FOREIGN KEY (cost_event_id, organization_id, business_unit_id) REFERENCES carrier_cost_events(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
ALTER TABLE carrier_assignment_accessorials
    ADD CONSTRAINT fk_carrier_assignment_accessorials_charge FOREIGN KEY (accessorial_charge_id, organization_id, business_unit_id) REFERENCES accessorial_charges(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
-- base_amount is the one computed column, so it joins the non-negative check.
ALTER TABLE carrier_assignments
    DROP CONSTRAINT IF EXISTS chk_carrier_assignments_amounts;

--bun:split
ALTER TABLE carrier_assignments
    ADD CONSTRAINT chk_carrier_assignments_amounts CHECK (base_rate >= 0 AND base_amount >= 0 AND fuel_surcharge >= 0 AND accessorial_total >= 0 AND total_cost >= 0);

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_cost_events_move ON carrier_cost_events(move_id, organization_id)
WHERE
    move_id IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_carrier_cost_events_pending ON carrier_cost_events(organization_id, business_unit_id, status, event_date);

--bun:split
-- Two system document types were both named "Rate Confirmation"; the outbound
-- carrier one is renamed so pickers can tell them apart.
UPDATE
    document_types
SET
    name = 'Carrier Rate Confirmation'
WHERE
    code = 'RATECON'
    AND is_system = TRUE
    AND name = 'Rate Confirmation';
