DROP TABLE IF EXISTS "customers" CASCADE;

-- bun:split

DROP TYPE IF EXISTS shipment_status_enum CASCADE;
DROP TYPE IF EXISTS rating_method_enum CASCADE;

-- bun:split

DROP INDEX IF EXISTS "shipments_pro_number_organization_id_unq" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_status" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_ship_date" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_bill_date" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_customer_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_origin_location_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_destination_location_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_created_by_id" CASCADE;
DROP INDEX IF EXISTS "idx_shipments_business_unit_id" CASCADE;
