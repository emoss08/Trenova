-- Drop index "qualifiercode_code_organization_id" from table: "qualifier_codes"
DROP INDEX "qualifiercode_code_organization_id";
-- Create index "qualifiercode_code_organization_id" to table: "qualifier_codes"
CREATE UNIQUE INDEX "qualifiercode_code_organization_id" ON "qualifier_codes" ("code", "organization_id");
-- Drop index "reasoncode_code_organization_id" from table: "reason_codes"
DROP INDEX "reasoncode_code_organization_id";
-- Create index "reasoncode_code_organization_id" to table: "reason_codes"
CREATE UNIQUE INDEX "reasoncode_code_organization_id" ON "reason_codes" ("code", "organization_id");
-- Drop index "tablechangealert_name_organization_id" from table: "table_change_alerts"
DROP INDEX "tablechangealert_name_organization_id";
-- Drop index "customer_code_organization_id" from table: "customers"
DROP INDEX "customer_code_organization_id";
-- Create index "customer_code_organization_id" to table: "customers"
CREATE UNIQUE INDEX "customer_code_organization_id" ON "customers" ("code", "organization_id");
-- Drop index "delaycode_code_organization_id" from table: "delay_codes"
DROP INDEX "delaycode_code_organization_id";
-- Create index "delaycode_code_organization_id" to table: "delay_codes"
CREATE UNIQUE INDEX "delaycode_code_organization_id" ON "delay_codes" ("code", "organization_id");
-- Drop index "divisioncode_code_organization_id" from table: "division_codes"
DROP INDEX "divisioncode_code_organization_id";
-- Create index "divisioncode_code_organization_id" to table: "division_codes"
CREATE UNIQUE INDEX "divisioncode_code_organization_id" ON "division_codes" ("code", "organization_id");
-- Drop index "documentclass_name_organization_id" from table: "document_classifications"
DROP INDEX "documentclass_name_organization_id";
-- Drop index "servicetype_code_organization_id" from table: "service_types"
DROP INDEX "servicetype_code_organization_id";
-- Create index "servicetype_code_organization_id" to table: "service_types"
CREATE UNIQUE INDEX "servicetype_code_organization_id" ON "service_types" ("code", "organization_id");
-- Drop index "commenttype_name_organization_id" from table: "comment_types"
DROP INDEX "commenttype_name_organization_id";
-- Create index "commenttype_name_organization_id" to table: "comment_types"
CREATE UNIQUE INDEX "commenttype_name_organization_id" ON "comment_types" ("name", "organization_id");
-- Drop index "revenuecode_code_organization_id" from table: "revenue_codes"
DROP INDEX "revenuecode_code_organization_id";
-- Create index "revenuecode_code_organization_id" to table: "revenue_codes"
CREATE UNIQUE INDEX "revenuecode_code_organization_id" ON "revenue_codes" ("code", "organization_id");
-- Drop index "locationcategory_name_organization_id" from table: "location_categories"
DROP INDEX "locationcategory_name_organization_id";
-- Create index "locationcategory_name_organization_id" to table: "location_categories"
CREATE UNIQUE INDEX "locationcategory_name_organization_id" ON "location_categories" ("name", "organization_id");
-- Drop index "generalledgeraccount_account_number_organization_id" from table: "general_ledger_accounts"
DROP INDEX "generalledgeraccount_account_number_organization_id";
-- Create index "generalledgeraccount_account_number_organization_id" to table: "general_ledger_accounts"
CREATE UNIQUE INDEX "generalledgeraccount_account_number_organization_id" ON "general_ledger_accounts" ("account_number", "organization_id");
-- Drop index "accessorialcharge_code_organization_id" from table: "accessorial_charges"
DROP INDEX "accessorialcharge_code_organization_id";
-- Create index "accessorialcharge_code_organization_id" to table: "accessorial_charges"
CREATE UNIQUE INDEX "accessorialcharge_code_organization_id" ON "accessorial_charges" ("code", "organization_id");
-- Drop index "chargetype_name_organization_id" from table: "charge_types"
DROP INDEX "chargetype_name_organization_id";
-- Create index "chargetype_name_organization_id" to table: "charge_types"
CREATE UNIQUE INDEX "chargetype_name_organization_id" ON "charge_types" ("name", "organization_id");
-- Drop index "equipmentmanufactuer_name_organization_id" from table: "equipment_manufactuers"
DROP INDEX "equipmentmanufactuer_name_organization_id";
-- Create index "equipmentmanufactuer_name_organization_id" to table: "equipment_manufactuers"
CREATE UNIQUE INDEX "equipmentmanufactuer_name_organization_id" ON "equipment_manufactuers" ("name", "organization_id");
-- Drop index "equipmenttype_name_organization_id" from table: "equipment_types"
DROP INDEX "equipmenttype_name_organization_id";
-- Create index "equipmenttype_name_organization_id" to table: "equipment_types"
CREATE UNIQUE INDEX "equipmenttype_name_organization_id" ON "equipment_types" ("name", "organization_id");
-- Drop index "fleetcode_code_organization_id" from table: "fleet_codes"
DROP INDEX "fleetcode_code_organization_id";
-- Create index "fleetcode_code_organization_id" to table: "fleet_codes"
CREATE UNIQUE INDEX "fleetcode_code_organization_id" ON "fleet_codes" ("code", "organization_id");
-- Create "workers" table
CREATE TABLE "workers" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "status" character varying NOT NULL DEFAULT 'A', "code" character varying NOT NULL, "profile_picture_url" character varying NULL, "worker_type" character varying NOT NULL DEFAULT 'Employee', "first_name" character varying NOT NULL, "last_name" character varying NOT NULL, "city" character varying NULL, "postal_code" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "state_id" uuid NULL, "fleet_code_id" uuid NULL, "manager_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "workers_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_fleet_codes_fleet_code" FOREIGN KEY ("fleet_code_id") REFERENCES "fleet_codes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_users_manager" FOREIGN KEY ("manager_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "worker_code_organization_id" to table: "workers"
CREATE UNIQUE INDEX "worker_code_organization_id" ON "workers" ("code", "organization_id");
-- Create index "worker_first_name_last_name" to table: "workers"
CREATE INDEX "worker_first_name_last_name" ON "workers" ("first_name", "last_name");
-- Create "tractors" table
CREATE TABLE "tractors" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "code" character varying NOT NULL, "status" character varying NOT NULL DEFAULT 'Available', "license_plate_number" character varying NULL, "vin" character varying NULL, "model" character varying NULL, "year" bigint NULL, "leased" boolean NOT NULL DEFAULT false, "leased_date" timestamptz NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "equipment_type_id" uuid NULL, "equipment_manufacturer_id" uuid NULL, "state_id" uuid NULL, "primary_worker_id" uuid NULL, "secondary_worker_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "tractors_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_equipment_manufactuers_equipment_manufacturer" FOREIGN KEY ("equipment_manufacturer_id") REFERENCES "equipment_manufactuers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_equipment_types_equipment_type" FOREIGN KEY ("equipment_type_id") REFERENCES "equipment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_workers_primary_worker" FOREIGN KEY ("primary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_workers_secondary_worker" FOREIGN KEY ("secondary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "tractor_code_organization_id" to table: "tractors"
CREATE UNIQUE INDEX "tractor_code_organization_id" ON "tractors" ("code", "organization_id");
