UPDATE
    document_types
SET
    name = 'Rate Confirmation'
WHERE
    code = 'RATECON'
    AND is_system = TRUE
    AND name = 'Carrier Rate Confirmation';

--bun:split
DROP INDEX IF EXISTS idx_carrier_cost_events_pending;

--bun:split
DROP INDEX IF EXISTS idx_carrier_cost_events_move;

--bun:split
ALTER TABLE carrier_assignments
    DROP CONSTRAINT IF EXISTS chk_carrier_assignments_amounts;

--bun:split
ALTER TABLE carrier_assignments
    ADD CONSTRAINT chk_carrier_assignments_amounts CHECK (base_rate >= 0 AND fuel_surcharge >= 0 AND accessorial_total >= 0 AND total_cost >= 0);

--bun:split
ALTER TABLE carrier_assignment_accessorials
    DROP CONSTRAINT IF EXISTS fk_carrier_assignment_accessorials_charge;

--bun:split
ALTER TABLE carrier_settlement_lines
    DROP CONSTRAINT IF EXISTS fk_carrier_settlement_lines_cost_event;

--bun:split
ALTER TABLE carrier_cost_events
    DROP CONSTRAINT IF EXISTS fk_carrier_cost_events_shipment_move,
    DROP CONSTRAINT IF EXISTS fk_carrier_cost_events_carrier_assignment;

--bun:split
ALTER TABLE carrier_cost_events
    DROP CONSTRAINT IF EXISTS fk_carrier_cost_events_shipment;

--bun:split
ALTER TABLE carrier_cost_events
    ADD CONSTRAINT fk_carrier_cost_events_shipment FOREIGN KEY (shipment_id, organization_id, business_unit_id) REFERENCES shipments(id, organization_id, business_unit_id) ON DELETE CASCADE;

--bun:split
ALTER TABLE carrier_invoice_matches
    DROP COLUMN IF EXISTS adjustment_cost_event_id;

--bun:split
DROP INDEX IF EXISTS uq_carrier_invoice_matches_extraction;

--bun:split
DROP INDEX IF EXISTS uq_carrier_settlements_carrier_period;

--bun:split
ALTER TABLE carrier_settlements
    DROP CONSTRAINT IF EXISTS fk_carrier_settlements_posted_expense_account,
    DROP CONSTRAINT IF EXISTS fk_carrier_settlements_posted_ap_account;

--bun:split
ALTER TABLE carrier_settlements
    DROP COLUMN IF EXISTS posted_expense_account_id,
    DROP COLUMN IF EXISTS posted_ap_account_id;
