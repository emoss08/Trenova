DROP TABLE IF EXISTS "shipment_moves" CASCADE;

-- bun:split

DROP TYPE IF EXISTS shipment_move_status_enum CASCADE;

-- bun:split

DROP INDEX IF EXISTS "idx_shipment_move_shipment_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_tractor_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_trailer_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_primary_worker_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipment_move_secondary_worker_id" CASCADE;
DROP INDEX IF EXISTS "idx_customer_org_bu" CASCADE;
DROP INDEX IF EXISTS "idx_customer_created_at" CASCADE;
