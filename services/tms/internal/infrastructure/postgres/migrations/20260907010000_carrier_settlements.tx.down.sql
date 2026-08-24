DROP TABLE IF EXISTS carrier_invoice_matches;

--bun:split
DROP TABLE IF EXISTS carrier_ledger_entries;

--bun:split
DROP TABLE IF EXISTS carrier_cost_events;

--bun:split
DROP TABLE IF EXISTS carrier_settlement_lines;

--bun:split
DROP TABLE IF EXISTS carrier_settlements;

--bun:split
DROP TABLE IF EXISTS carrier_settlement_batches;

--bun:split
DROP TABLE IF EXISTS carrier_settlement_controls;

--bun:split
DROP INDEX IF EXISTS idx_edi_carrier_invoices_carrier;

--bun:split
ALTER TABLE edi_carrier_invoices
    DROP CONSTRAINT IF EXISTS fk_edi_carrier_invoices_carrier;

--bun:split
ALTER TABLE edi_carrier_invoices
    DROP COLUMN IF EXISTS carrier_id;
