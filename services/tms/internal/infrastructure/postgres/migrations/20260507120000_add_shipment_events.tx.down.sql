DROP INDEX IF EXISTS idx_shipment_events_type_occurred;

--bun:split
DROP INDEX IF EXISTS idx_shipment_events_shipment_occurred;

--bun:split
DROP INDEX IF EXISTS idx_shipment_events_tenant_occurred;

--bun:split
DROP TABLE IF EXISTS shipment_events;
