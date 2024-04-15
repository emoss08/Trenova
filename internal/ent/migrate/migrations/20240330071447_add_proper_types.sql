-- Modify "hazardous_material_segregations" table
ALTER TABLE "hazardous_material_segregations" ALTER COLUMN "class_a" TYPE character varying(16), ALTER COLUMN "class_b" TYPE character varying(16), ALTER COLUMN "segregation_type" TYPE character varying(21);
-- Modify "comment_types" table
ALTER TABLE "comment_types" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "name" TYPE character varying(10), ALTER COLUMN "severity" TYPE character varying(6);
-- Modify "accessorial_charges" table
ALTER TABLE "accessorial_charges" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "code" TYPE character varying(4), ALTER COLUMN "description" TYPE text, ALTER COLUMN "method" TYPE character varying(10);
-- Modify "charge_types" table
ALTER TABLE "charge_types" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "name" TYPE character varying(50), ALTER COLUMN "description" TYPE text;
-- Modify "hazardous_materials" table
ALTER TABLE "hazardous_materials" ALTER COLUMN "name" TYPE character varying(100), ALTER COLUMN "hazard_class" TYPE character varying(16);
-- Modify "commodities" table
ALTER TABLE "commodities" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "name" TYPE character varying(100), ALTER COLUMN "min_temp" TYPE smallint, ALTER COLUMN "max_temp" TYPE smallint;
-- Modify "workers" table
ALTER TABLE "workers" ALTER COLUMN "code" TYPE character varying(10), ALTER COLUMN "postal_code" TYPE character varying(10);
-- Modify "delay_codes" table
ALTER TABLE "delay_codes" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "code" TYPE character varying(4);
-- Modify "dispatch_controls" table
ALTER TABLE "dispatch_controls" ALTER COLUMN "record_service_incident" TYPE character varying(17), ALTER COLUMN "max_shipment_weight_limit" TYPE integer;
-- Modify "division_codes" table
ALTER TABLE "division_codes" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "code" TYPE character varying(4);
-- Modify "document_classifications" table
ALTER TABLE "document_classifications" ALTER COLUMN "name" TYPE character varying(10), ALTER COLUMN "status" TYPE character varying(1);
-- Modify "invoice_controls" table
ALTER TABLE "invoice_controls" ALTER COLUMN "invoice_number_prefix" TYPE character varying(10), ALTER COLUMN "credit_memo_number_prefix" TYPE character varying(10);
-- Modify "equipment_manufactuers" table
ALTER TABLE "equipment_manufactuers" ALTER COLUMN "status" TYPE character varying(1);
-- Modify "equipment_types" table
ALTER TABLE "equipment_types" ALTER COLUMN "equipment_class" TYPE character varying(10);
-- Modify "billing_controls" table
ALTER TABLE "billing_controls" ALTER COLUMN "shipment_transfer_criteria" TYPE character varying(17);
-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" ALTER COLUMN "rec_threshold" TYPE smallint;
-- Modify "email_profiles" table
ALTER TABLE "email_profiles" ALTER COLUMN "name" TYPE character varying(150), ALTER COLUMN "protocol" TYPE character varying(11), ALTER COLUMN "port" TYPE smallint;
-- Modify "location_categories" table
ALTER TABLE "location_categories" ALTER COLUMN "name" TYPE character varying(100);
-- Modify "organizations" table
ALTER TABLE "organizations" ALTER COLUMN "name" TYPE character varying(100), ALTER COLUMN "scac_code" TYPE character varying(4), ALTER COLUMN "dot_number" TYPE character varying(12), ALTER COLUMN "org_type" TYPE character varying(1), ALTER COLUMN "timezone" TYPE character varying(17);
-- Modify "qualifier_codes" table
ALTER TABLE "qualifier_codes" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "code" TYPE character varying(10);
-- Modify "reason_codes" table
ALTER TABLE "reason_codes" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "code" TYPE character varying(10);
-- Modify "revenue_codes" table
ALTER TABLE "revenue_codes" ALTER COLUMN "code" TYPE character varying(4), ALTER COLUMN "status" TYPE character varying(1);
-- Modify "route_controls" table
ALTER TABLE "route_controls" ALTER COLUMN "distance_method" TYPE character varying(1), ALTER COLUMN "mileage_unit" TYPE character varying(1);
-- Modify "service_types" table
ALTER TABLE "service_types" ALTER COLUMN "code" TYPE character varying(10);
-- Modify "shipment_types" table
ALTER TABLE "shipment_types" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "code" TYPE character varying(10);
-- Modify "table_change_alerts" table
ALTER TABLE "table_change_alerts" ALTER COLUMN "name" TYPE character varying(50), ALTER COLUMN "database_action" TYPE character varying(6), ALTER COLUMN "function_name" TYPE character varying(50), ALTER COLUMN "trigger_name" TYPE character varying(50), ALTER COLUMN "listener_name" TYPE character varying(50), ALTER COLUMN "effective_date" TYPE date, ALTER COLUMN "expiration_date" TYPE date;
-- Modify "tags" table
ALTER TABLE "tags" ALTER COLUMN "name" TYPE character varying(50);
-- Modify "users" table
ALTER TABLE "users" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "timezone" TYPE character varying(17);
-- Modify "customers" table
ALTER TABLE "customers" ALTER COLUMN "status" TYPE character varying(1), ALTER COLUMN "code" TYPE character varying(10), ALTER COLUMN "name" TYPE character varying(150), ALTER COLUMN "address_line_1" TYPE character varying(150), ALTER COLUMN "address_line_2" TYPE character varying(150), ALTER COLUMN "city" TYPE character varying(150), DROP COLUMN "state", ALTER COLUMN "postal_code" TYPE character varying(10), ADD COLUMN "state_id" uuid NULL, ADD CONSTRAINT "customers_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
