-- Modify "accessorial_charges" table
ALTER TABLE "accessorial_charges" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" DROP COLUMN "version";
-- Modify "billing_controls" table
ALTER TABLE "billing_controls" DROP COLUMN "version";
-- Modify "business_units" table
ALTER TABLE "business_units" DROP COLUMN "version";
-- Modify "charge_types" table
ALTER TABLE "charge_types" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "comment_types" table
ALTER TABLE "comment_types" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "commodities" table
ALTER TABLE "commodities" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "customers" table
ALTER TABLE "customers" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "delay_codes" table
ALTER TABLE "delay_codes" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "dispatch_controls" table
ALTER TABLE "dispatch_controls" DROP COLUMN "version";
-- Modify "division_codes" table
ALTER TABLE "division_codes" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "document_classifications" table
ALTER TABLE "document_classifications" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "email_controls" table
ALTER TABLE "email_controls" DROP COLUMN "version";
-- Modify "email_profiles" table
ALTER TABLE "email_profiles" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "equipment_manufactuers" table
ALTER TABLE "equipment_manufactuers" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "equipment_types" table
ALTER TABLE "equipment_types" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "feasibility_tool_controls" table
ALTER TABLE "feasibility_tool_controls" DROP COLUMN "version";
-- Modify "fleet_codes" table
ALTER TABLE "fleet_codes" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "general_ledger_accounts" table
ALTER TABLE "general_ledger_accounts" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "google_apis" table
ALTER TABLE "google_apis" DROP COLUMN "version";
-- Modify "hazardous_material_segregations" table
ALTER TABLE "hazardous_material_segregations" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "hazardous_materials" table
ALTER TABLE "hazardous_materials" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "invoice_controls" table
ALTER TABLE "invoice_controls" DROP COLUMN "version";
-- Modify "location_categories" table
ALTER TABLE "location_categories" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "organizations" table
ALTER TABLE "organizations" DROP COLUMN "version";
-- Modify "qualifier_codes" table
ALTER TABLE "qualifier_codes" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "reason_codes" table
ALTER TABLE "reason_codes" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "revenue_codes" table
ALTER TABLE "revenue_codes" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "route_controls" table
ALTER TABLE "route_controls" DROP COLUMN "version";
-- Modify "service_types" table
ALTER TABLE "service_types" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "shipment_controls" table
ALTER TABLE "shipment_controls" DROP COLUMN "version";
-- Modify "shipment_types" table
ALTER TABLE "shipment_types" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "table_change_alerts" table
ALTER TABLE "table_change_alerts" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "tags" table
ALTER TABLE "tags" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "tractors" table
ALTER TABLE "tractors" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "us_states" table
ALTER TABLE "us_states" DROP COLUMN "version";
-- Modify "user_favorites" table
ALTER TABLE "user_favorites" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "users" table
ALTER TABLE "users" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "worker_comments" table
ALTER TABLE "worker_comments" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "worker_contacts" table
ALTER TABLE "worker_contacts" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "worker_profiles" table
ALTER TABLE "worker_profiles" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "workers" table
ALTER TABLE "workers" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
