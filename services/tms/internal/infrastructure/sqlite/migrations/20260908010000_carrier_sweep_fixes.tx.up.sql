-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260908010000_carrier_sweep_fixes.tx.up.sql

ALTER TABLE "carrier_settlements" ADD COLUMN "posted_expense_account_id" TEXT;

--bun:split

ALTER TABLE "carrier_settlements" ADD COLUMN "posted_ap_account_id" TEXT;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_settlements_carrier_period ON carrier_settlements (organization_id, business_unit_id, carrier_id, period_start, period_end)WHERE
    status <> 'Voided';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_invoice_matches_extraction ON carrier_invoice_matches (document_ai_extraction_id, organization_id, business_unit_id)WHERE
    document_ai_extraction_id IS NOT NULL AND status <> 'Rejected';

--bun:split

CREATE INDEX IF NOT EXISTS idx_carrier_invoice_matches_adjustment_event ON carrier_invoice_matches (adjustment_cost_event_id, organization_id)WHERE
    adjustment_cost_event_id IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_carrier_cost_events_move ON carrier_cost_events (move_id, organization_id)WHERE
    move_id IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_carrier_cost_events_pending ON carrier_cost_events (organization_id, business_unit_id, status, event_date);

--bun:split

UPDATE
    document_types
SET
    name = 'Carrier Rate Confirmation'
WHERE
    code = 'RATECON'
    AND is_system = TRUE
    AND name = 'Rate Confirmation';
