-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "billing_controls" table
ALTER TABLE "billing_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "business_units" table
ALTER TABLE "business_units" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "comment_types" table
ALTER TABLE "comment_types" DROP COLUMN "version";
-- Modify "dispatch_controls" table
ALTER TABLE "dispatch_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "email_controls" table
ALTER TABLE "email_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "feasibility_tool_controls" table
ALTER TABLE "feasibility_tool_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "google_apis" table
ALTER TABLE "google_apis" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "invoice_controls" table
ALTER TABLE "invoice_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "organizations" table
ALTER TABLE "organizations" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "route_controls" table
ALTER TABLE "route_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "shipment_controls" table
ALTER TABLE "shipment_controls" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
-- Modify "us_states" table
ALTER TABLE "us_states" ADD COLUMN "version" bigint NOT NULL DEFAULT 1;
