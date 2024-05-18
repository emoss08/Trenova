-- Create "sessions" table
CREATE TABLE "sessions" ("id" character varying NOT NULL, "data" character varying NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "expires_at" timestamptz NOT NULL, PRIMARY KEY ("id"));
-- Create "business_units" table
CREATE TABLE "business_units" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "status" character varying NOT NULL DEFAULT 'A', "name" character varying NOT NULL, "entity_key" character varying NOT NULL, "phone_number" character varying NOT NULL, "address" character varying NULL, "city" character varying NULL, "state" character varying NULL, "country" character varying NULL, "postal_code" character varying NULL, "tax_id" character varying NULL, "subscription_plan" character varying NULL, "description" text NULL, "legal_name" character varying NULL, "contact_name" character varying NULL, "contact_email" character varying NULL, "paid_until" timestamptz NULL, "settings" jsonb NULL, "free_trial" boolean NOT NULL DEFAULT false, "parent_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "business_units_business_units_next" FOREIGN KEY ("parent_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE SET NULL);
-- Create index "business_units_parent_id_key" to table: "business_units"
CREATE UNIQUE INDEX "business_units_parent_id_key" ON "business_units" ("parent_id");
-- Create index "businessunit_entity_key" to table: "business_units"
CREATE UNIQUE INDEX "businessunit_entity_key" ON "business_units" ("entity_key");
-- Create index "businessunit_name" to table: "business_units"
CREATE UNIQUE INDEX "businessunit_name" ON "business_units" ("name");
-- Create "organizations" table
CREATE TABLE "organizations" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "name" character varying(100) NOT NULL, "scac_code" character varying(4) NOT NULL, "dot_number" character varying(12) NOT NULL, "logo_url" character varying NULL, "org_type" character varying(1) NOT NULL DEFAULT 'A', "timezone" character varying(20) NOT NULL, "business_unit_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "organizations_business_units_organizations" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create "accessorial_charges" table
CREATE TABLE "accessorial_charges" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(4) NOT NULL, "description" text NULL, "is_detention" boolean NOT NULL DEFAULT false, "method" character varying(10) NOT NULL, "amount" numeric(19,4) NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "accessorial_charges_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "accessorial_charges_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "accessorial_charges"
COMMENT ON COLUMN "accessorial_charges" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "accessorial_charges"
COMMENT ON COLUMN "accessorial_charges" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "accessorial_charges"
COMMENT ON COLUMN "accessorial_charges" ."version" IS 'The current version of this entity.';
-- Create "general_ledger_accounts" table
CREATE TABLE "general_ledger_accounts" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "account_number" character varying(7) NOT NULL, "account_type" character varying(9) NOT NULL, "cash_flow_type" character varying NULL, "account_sub_type" character varying NULL, "account_class" character varying NULL, "balance" double precision NULL, "interest_rate" double precision NULL, "date_closed" date NULL, "notes" character varying NULL, "is_tax_relevant" boolean NOT NULL DEFAULT false, "is_reconciled" boolean NOT NULL DEFAULT false, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "general_ledger_accounts_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "general_ledger_accounts_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "general_ledger_accounts"
COMMENT ON COLUMN "general_ledger_accounts" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "general_ledger_accounts"
COMMENT ON COLUMN "general_ledger_accounts" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "general_ledger_accounts"
COMMENT ON COLUMN "general_ledger_accounts" ."version" IS 'The current version of this entity.';
-- Create "accounting_controls" table
CREATE TABLE "accounting_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "rec_threshold" smallint NOT NULL DEFAULT 50, "rec_threshold_action" character varying NOT NULL DEFAULT 'Halt', "auto_create_journal_entries" boolean NOT NULL DEFAULT false, "journal_entry_criteria" character varying NOT NULL DEFAULT 'OnShipmentBill', "restrict_manual_journal_entries" boolean NOT NULL DEFAULT false, "require_journal_entry_approval" boolean NOT NULL DEFAULT false, "enable_rec_notifications" boolean NOT NULL DEFAULT true, "halt_on_pending_rec" boolean NOT NULL DEFAULT false, "critical_processes" text NULL, "business_unit_id" uuid NOT NULL, "default_rev_account_id" uuid NULL, "default_exp_account_id" uuid NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "accounting_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "accounting_controls_general_ledger_accounts_default_exp_account" FOREIGN KEY ("default_exp_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "accounting_controls_general_ledger_accounts_default_rev_account" FOREIGN KEY ("default_rev_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "accounting_controls_organizations_accounting_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "accounting_controls_organization_id_key" to table: "accounting_controls"
CREATE UNIQUE INDEX "accounting_controls_organization_id_key" ON "accounting_controls" ("organization_id");
-- Create "billing_controls" table
CREATE TABLE "billing_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "remove_billing_history" boolean NOT NULL DEFAULT false, "auto_bill_shipment" boolean NOT NULL DEFAULT false, "auto_mark_ready_to_bill" boolean NOT NULL DEFAULT false, "validate_customer_rates" boolean NOT NULL DEFAULT false, "auto_bill_criteria" character varying NOT NULL DEFAULT 'MarkedReadyToBill', "shipment_transfer_criteria" character varying(17) NOT NULL DEFAULT 'ReadyToBill', "enforce_customer_billing" boolean NOT NULL DEFAULT false, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "billing_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "billing_controls_organizations_billing_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "billing_controls_organization_id_key" to table: "billing_controls"
CREATE UNIQUE INDEX "billing_controls_organization_id_key" ON "billing_controls" ("organization_id");
-- Create "charge_types" table
CREATE TABLE "charge_types" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "name" character varying(50) NOT NULL, "description" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "charge_types_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "charge_types_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "charge_types"
COMMENT ON COLUMN "charge_types" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "charge_types"
COMMENT ON COLUMN "charge_types" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "charge_types"
COMMENT ON COLUMN "charge_types" ."version" IS 'The current version of this entity.';
-- Create "comment_types" table
CREATE TABLE "comment_types" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "name" character varying(20) NOT NULL, "severity" character varying(6) NOT NULL DEFAULT 'Low', "description" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "comment_types_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "comment_types_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "comment_types"
COMMENT ON COLUMN "comment_types" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "comment_types"
COMMENT ON COLUMN "comment_types" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "comment_types"
COMMENT ON COLUMN "comment_types" ."version" IS 'The current version of this entity.';
-- Create "hazardous_materials" table
CREATE TABLE "hazardous_materials" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'A', "name" character varying(100) NOT NULL, "hazard_class" character varying(16) NOT NULL DEFAULT 'HazardClass1And1', "erg_number" character varying NULL, "description" text NULL, "packing_group" character varying NULL, "proper_shipping_name" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "hazardous_materials_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "hazardous_materials_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "hazardous_materials"
COMMENT ON COLUMN "hazardous_materials" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "hazardous_materials"
COMMENT ON COLUMN "hazardous_materials" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "hazardous_materials"
COMMENT ON COLUMN "hazardous_materials" ."version" IS 'The current version of this entity.';
-- Create "commodities" table
CREATE TABLE "commodities" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "name" character varying(100) NOT NULL, "is_hazmat" boolean NOT NULL DEFAULT false, "unit_of_measure" character varying NULL, "min_temp" smallint NULL, "max_temp" smallint NULL, "description" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "hazardous_material_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "commodities_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "commodities_hazardous_materials_hazardous_material" FOREIGN KEY ("hazardous_material_id") REFERENCES "hazardous_materials" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT, CONSTRAINT "commodities_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "commodities"
COMMENT ON COLUMN "commodities" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "commodities"
COMMENT ON COLUMN "commodities" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "commodities"
COMMENT ON COLUMN "commodities" ."version" IS 'The current version of this entity.';
-- Create "custom_reports" table
CREATE TABLE "custom_reports" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying NOT NULL, "description" character varying NULL, "table" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "custom_reports_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "custom_reports_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "custom_reports"
COMMENT ON COLUMN "custom_reports" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "custom_reports"
COMMENT ON COLUMN "custom_reports" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "custom_reports"
COMMENT ON COLUMN "custom_reports" ."version" IS 'The current version of this entity.';
-- Create "us_states" table
CREATE TABLE "us_states" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "name" character varying NOT NULL, "abbreviation" character varying NOT NULL, "country_name" character varying NOT NULL DEFAULT 'United States', "country_iso3" character varying NOT NULL DEFAULT 'USA', PRIMARY KEY ("id"));
-- Create "customers" table
CREATE TABLE "customers" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "name" character varying(150) NOT NULL, "address_line_1" character varying(150) NOT NULL, "address_line_2" character varying(150) NULL, "city" character varying(150) NOT NULL, "postal_code" character varying(10) NOT NULL, "has_customer_portal" boolean NOT NULL DEFAULT false, "auto_mark_ready_to_bill" boolean NOT NULL DEFAULT false, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "state_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "customers_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customers_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customers_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT);
-- Create index "customers_code_key" to table: "customers"
CREATE UNIQUE INDEX "customers_code_key" ON "customers" ("code");
-- Set comment to column: "created_at" on table: "customers"
COMMENT ON COLUMN "customers" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "customers"
COMMENT ON COLUMN "customers" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "customers"
COMMENT ON COLUMN "customers" ."version" IS 'The current version of this entity.';
-- Create "customer_contacts" table
CREATE TABLE "customer_contacts" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying(150) NOT NULL, "email" character varying(150) NULL, "title" character varying(100) NULL, "phone_number" character varying(15) NULL, "is_payable_contact" boolean NOT NULL DEFAULT false, "customer_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "customer_contacts_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_contacts_customers_contacts" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_contacts_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "customer_contacts"
COMMENT ON COLUMN "customer_contacts" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "customer_contacts"
COMMENT ON COLUMN "customer_contacts" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "customer_contacts"
COMMENT ON COLUMN "customer_contacts" ."version" IS 'The current version of this entity.';
-- Create "revenue_codes" table
CREATE TABLE "revenue_codes" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(4) NOT NULL, "description" text NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "expense_account_id" uuid NULL, "revenue_account_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "revenue_codes_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "revenue_codes_general_ledger_accounts_expense_account" FOREIGN KEY ("expense_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "revenue_codes_general_ledger_accounts_revenue_account" FOREIGN KEY ("revenue_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "revenue_codes_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "revenue_codes"
COMMENT ON COLUMN "revenue_codes" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "revenue_codes"
COMMENT ON COLUMN "revenue_codes" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "revenue_codes"
COMMENT ON COLUMN "revenue_codes" ."version" IS 'The current version of this entity.';
-- Create "shipment_types" table
CREATE TABLE "shipment_types" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "description" text NULL, "color" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_types_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_types_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "shipment_types"
COMMENT ON COLUMN "shipment_types" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipment_types"
COMMENT ON COLUMN "shipment_types" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipment_types"
COMMENT ON COLUMN "shipment_types" ."version" IS 'The current version of this entity.';
-- Create "customer_detention_policies" table
CREATE TABLE "customer_detention_policies" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "application_scope" character varying NULL DEFAULT 'PICKUP', "charge_free_time" bigint NULL, "payment_free_time" bigint NULL, "late_arrival_policy" boolean NULL DEFAULT false, "grace_period" bigint NULL, "units" bigint NULL, "amount" numeric(19,4) NOT NULL, "notes" text NULL, "effective_date" date NULL, "expiration_date" date NULL, "customer_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "commodity_id" uuid NULL, "revenue_code_id" uuid NULL, "shipment_type_id" uuid NULL, "accessorial_charge_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "customer_detention_policies_accessorial_charges_accessorial_cha" FOREIGN KEY ("accessorial_charge_id") REFERENCES "accessorial_charges" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "customer_detention_policies_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_detention_policies_commodities_commodity" FOREIGN KEY ("commodity_id") REFERENCES "commodities" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "customer_detention_policies_customers_detention_policies" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "customer_detention_policies_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_detention_policies_revenue_codes_revenue_code" FOREIGN KEY ("revenue_code_id") REFERENCES "revenue_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "customer_detention_policies_shipment_types_shipment_type" FOREIGN KEY ("shipment_type_id") REFERENCES "shipment_types" ("id") ON UPDATE NO ACTION ON DELETE SET NULL);
-- Set comment to column: "created_at" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."version" IS 'The current version of this entity.';
-- Set comment to column: "application_scope" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."application_scope" IS 'Specifies whether the policy applies to pickups, deliveries, or both.';
-- Set comment to column: "charge_free_time" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."charge_free_time" IS 'The threshold time (in minutes) for the start of detention charges. This represents the allowed free time before charges apply.';
-- Set comment to column: "payment_free_time" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."payment_free_time" IS 'The time (in minutes) considered for calculating detention payments. This can differ from charge_free_time in certain scenarios.';
-- Set comment to column: "late_arrival_policy" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."late_arrival_policy" IS 'Indicates whether the policy applies to late arrivals. True if detention charges apply to late arrivals.';
-- Set comment to column: "grace_period" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."grace_period" IS 'An additional time buffer (in minutes) provided before detention charges kick in, often used to accommodate slight delays.';
-- Set comment to column: "units" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."units" IS 'The number of units (e.g., pallets, containers) considered for detention charges.';
-- Set comment to column: "notes" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."notes" IS 'Additional notes or comments about the detention policy.';
-- Set comment to column: "effective_date" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."effective_date" IS 'The date when the detention policy becomes effective.';
-- Set comment to column: "expiration_date" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."expiration_date" IS 'The date when the detention policy expires.';
-- Set comment to column: "commodity_id" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."commodity_id" IS 'The type of commodity to which the detention policy applies. This helps in customizing policies for different commodities.';
-- Set comment to column: "revenue_code_id" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."revenue_code_id" IS 'A unique code associated with the revenue generated from detention charges.';
-- Set comment to column: "shipment_type_id" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."shipment_type_id" IS 'Type of shipment (e.g., Standard, Expedited) to which the detention policy is applicable.';
-- Set comment to column: "accessorial_charge_id" on table: "customer_detention_policies"
COMMENT ON COLUMN "customer_detention_policies" ."accessorial_charge_id" IS 'The unique identifier for the accessorial charge associated with the detention policy.';
-- Create "email_profiles" table
CREATE TABLE "email_profiles" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying(150) NOT NULL, "email" character varying NOT NULL, "protocol" character varying(11) NULL, "host" character varying NULL, "port" smallint NULL, "username" character varying NULL, "password" character varying NULL, "is_default" boolean NOT NULL DEFAULT false, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "email_profiles_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "email_profiles_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "email_profiles"
COMMENT ON COLUMN "email_profiles" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "email_profiles"
COMMENT ON COLUMN "email_profiles" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "email_profiles"
COMMENT ON COLUMN "email_profiles" ."version" IS 'The current version of this entity.';
-- Create "customer_email_profiles" table
CREATE TABLE "customer_email_profiles" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "subject" character varying(100) NULL, "email_recipients" text NOT NULL, "email_cc_recipients" text NULL, "attachment_name" text NULL, "email_format" character varying NOT NULL DEFAULT 'PLAIN', "customer_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "email_profile_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "customer_email_profiles_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_email_profiles_customers_email_profile" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_email_profiles_email_profiles_email_profile" FOREIGN KEY ("email_profile_id") REFERENCES "email_profiles" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "customer_email_profiles_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "customer_email_profiles_customer_id_key" to table: "customer_email_profiles"
CREATE UNIQUE INDEX "customer_email_profiles_customer_id_key" ON "customer_email_profiles" ("customer_id");
-- Set comment to column: "created_at" on table: "customer_email_profiles"
COMMENT ON COLUMN "customer_email_profiles" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "customer_email_profiles"
COMMENT ON COLUMN "customer_email_profiles" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "customer_email_profiles"
COMMENT ON COLUMN "customer_email_profiles" ."version" IS 'The current version of this entity.';
-- Create "customer_rule_profiles" table
CREATE TABLE "customer_rule_profiles" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "billing_cycle" character varying NOT NULL DEFAULT 'PER_SHIPMENT', "customer_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "customer_rule_profiles_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_rule_profiles_customers_rule_profile" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "customer_rule_profiles_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "customer_rule_profiles_customer_id_key" to table: "customer_rule_profiles"
CREATE UNIQUE INDEX "customer_rule_profiles_customer_id_key" ON "customer_rule_profiles" ("customer_id");
-- Set comment to column: "created_at" on table: "customer_rule_profiles"
COMMENT ON COLUMN "customer_rule_profiles" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "customer_rule_profiles"
COMMENT ON COLUMN "customer_rule_profiles" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "customer_rule_profiles"
COMMENT ON COLUMN "customer_rule_profiles" ."version" IS 'The current version of this entity.';
-- Create "delay_codes" table
CREATE TABLE "delay_codes" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(20) NOT NULL, "description" text NULL, "f_carrier_or_driver" boolean NULL, "color" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "delay_codes_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "delay_codes_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "delay_codes"
COMMENT ON COLUMN "delay_codes" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "delay_codes"
COMMENT ON COLUMN "delay_codes" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "delay_codes"
COMMENT ON COLUMN "delay_codes" ."version" IS 'The current version of this entity.';
-- Create "location_categories" table
CREATE TABLE "location_categories" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying(100) NOT NULL, "description" text NULL, "color" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "location_categories_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "location_categories_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "location_categories"
COMMENT ON COLUMN "location_categories" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "location_categories"
COMMENT ON COLUMN "location_categories" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "location_categories"
COMMENT ON COLUMN "location_categories" ."version" IS 'The current version of this entity.';
-- Create "locations" table
CREATE TABLE "locations" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'A', "code" character varying NOT NULL, "name" character varying NOT NULL, "description" text NULL, "address_line_1" character varying(150) NOT NULL, "address_line_2" character varying(150) NULL, "city" character varying(150) NOT NULL, "postal_code" character varying(10) NOT NULL, "longitude" double precision NULL, "latitude" double precision NULL, "place_id" character varying(255) NULL, "is_geocoded" boolean NOT NULL DEFAULT false, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "location_category_id" uuid NULL, "state_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "locations_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "locations_location_categories_location_category" FOREIGN KEY ("location_category_id") REFERENCES "location_categories" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "locations_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "locations_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "locations"
COMMENT ON COLUMN "locations" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "locations"
COMMENT ON COLUMN "locations" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "locations"
COMMENT ON COLUMN "locations" ."version" IS 'The current version of this entity.';
-- Set comment to column: "status" on table: "locations"
COMMENT ON COLUMN "locations" ."status" IS 'Current status of the location.';
-- Set comment to column: "code" on table: "locations"
COMMENT ON COLUMN "locations" ."code" IS 'Unique code for the location.';
-- Set comment to column: "name" on table: "locations"
COMMENT ON COLUMN "locations" ."name" IS 'Name of the location.';
-- Set comment to column: "description" on table: "locations"
COMMENT ON COLUMN "locations" ."description" IS 'Description of the location.';
-- Set comment to column: "address_line_1" on table: "locations"
COMMENT ON COLUMN "locations" ."address_line_1" IS 'Adress Line 1 of the location.';
-- Set comment to column: "address_line_2" on table: "locations"
COMMENT ON COLUMN "locations" ."address_line_2" IS 'Adress Line 2 of the location.';
-- Set comment to column: "city" on table: "locations"
COMMENT ON COLUMN "locations" ."city" IS 'City of the location.';
-- Set comment to column: "postal_code" on table: "locations"
COMMENT ON COLUMN "locations" ."postal_code" IS 'Postal code of the location.';
-- Set comment to column: "longitude" on table: "locations"
COMMENT ON COLUMN "locations" ."longitude" IS 'Longitude of the location.';
-- Set comment to column: "latitude" on table: "locations"
COMMENT ON COLUMN "locations" ."latitude" IS 'Latitude of the location.';
-- Set comment to column: "place_id" on table: "locations"
COMMENT ON COLUMN "locations" ."place_id" IS 'Place ID from Google Maps API.';
-- Set comment to column: "is_geocoded" on table: "locations"
COMMENT ON COLUMN "locations" ."is_geocoded" IS 'Is the location geocoded?';
-- Set comment to column: "location_category_id" on table: "locations"
COMMENT ON COLUMN "locations" ."location_category_id" IS 'Location category ID.';
-- Set comment to column: "state_id" on table: "locations"
COMMENT ON COLUMN "locations" ."state_id" IS 'State ID.';
-- Create "delivery_slots" table
CREATE TABLE "delivery_slots" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "day_of_week" character varying NOT NULL, "start_time" time NOT NULL, "end_time" time NOT NULL, "customer_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "location_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "delivery_slots_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "delivery_slots_customers_delivery_slots" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "delivery_slots_locations_location" FOREIGN KEY ("location_id") REFERENCES "locations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "delivery_slots_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "valid_start_time_end_time" CHECK (start_time < end_time));
-- Set comment to column: "created_at" on table: "delivery_slots"
COMMENT ON COLUMN "delivery_slots" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "delivery_slots"
COMMENT ON COLUMN "delivery_slots" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "delivery_slots"
COMMENT ON COLUMN "delivery_slots" ."version" IS 'The current version of this entity.';
-- Create "dispatch_controls" table
CREATE TABLE "dispatch_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "record_service_incident" character varying(17) NOT NULL DEFAULT 'Never', "deadhead_target" double precision NOT NULL DEFAULT 0, "max_shipment_weight_limit" integer NOT NULL DEFAULT 80000, "grace_period" smallint NOT NULL DEFAULT 0, "enforce_worker_assign" boolean NOT NULL DEFAULT true, "trailer_continuity" boolean NOT NULL DEFAULT false, "dupe_trailer_check" boolean NOT NULL DEFAULT false, "maintenance_compliance" boolean NOT NULL DEFAULT true, "regulatory_check" boolean NOT NULL DEFAULT false, "prev_shipment_on_hold" boolean NOT NULL DEFAULT false, "worker_time_away_restriction" boolean NOT NULL DEFAULT true, "tractor_worker_fleet_constraint" boolean NOT NULL DEFAULT false, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "dispatch_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "dispatch_controls_organizations_dispatch_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "dispatch_controls_organization_id_key" to table: "dispatch_controls"
CREATE UNIQUE INDEX "dispatch_controls_organization_id_key" ON "dispatch_controls" ("organization_id");
-- Create "division_codes" table
CREATE TABLE "division_codes" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(4) NOT NULL, "description" text NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "cash_account_id" uuid NULL, "ap_account_id" uuid NULL, "expense_account_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "division_codes_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "division_codes_general_ledger_accounts_ap_account" FOREIGN KEY ("ap_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "division_codes_general_ledger_accounts_cash_account" FOREIGN KEY ("cash_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "division_codes_general_ledger_accounts_expense_account" FOREIGN KEY ("expense_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "division_codes_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "division_codes"
COMMENT ON COLUMN "division_codes" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "division_codes"
COMMENT ON COLUMN "division_codes" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "division_codes"
COMMENT ON COLUMN "division_codes" ."version" IS 'The current version of this entity.';
-- Create "document_classifications" table
CREATE TABLE "document_classifications" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "description" text NULL, "color" character varying NULL, "customer_rule_profile_document_classifications" uuid NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "document_classifications_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "document_classifications_customer_rule_profiles_document_classi" FOREIGN KEY ("customer_rule_profile_document_classifications") REFERENCES "customer_rule_profiles" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT "document_classifications_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "document_classifications"
COMMENT ON COLUMN "document_classifications" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "document_classifications"
COMMENT ON COLUMN "document_classifications" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "document_classifications"
COMMENT ON COLUMN "document_classifications" ."version" IS 'The current version of this entity.';
-- Create "email_controls" table
CREATE TABLE "email_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "business_unit_id" uuid NOT NULL, "billing_email_profile_id" uuid NULL, "rate_expirtation_email_profile_id" uuid NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "email_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "email_controls_email_profiles_billing_email_profile" FOREIGN KEY ("billing_email_profile_id") REFERENCES "email_profiles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "email_controls_email_profiles_rate_email_profile" FOREIGN KEY ("rate_expirtation_email_profile_id") REFERENCES "email_profiles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "email_controls_organizations_email_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "email_controls_organization_id_key" to table: "email_controls"
CREATE UNIQUE INDEX "email_controls_organization_id_key" ON "email_controls" ("organization_id");
-- Create "equipment_manufactuers" table
CREATE TABLE "equipment_manufactuers" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "name" character varying NOT NULL, "description" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "equipment_manufactuers_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "equipment_manufactuers_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "equipment_manufactuers"
COMMENT ON COLUMN "equipment_manufactuers" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "equipment_manufactuers"
COMMENT ON COLUMN "equipment_manufactuers" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "equipment_manufactuers"
COMMENT ON COLUMN "equipment_manufactuers" ."version" IS 'The current version of this entity.';
-- Create "equipment_types" table
CREATE TABLE "equipment_types" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "description" text NULL, "cost_per_mile" numeric(10,2) NULL, "equipment_class" character varying(10) NOT NULL DEFAULT 'Undefined', "fixed_cost" numeric(10,2) NULL, "variable_cost" numeric(10,2) NULL, "height" numeric(10,2) NULL, "length" numeric(10,2) NULL, "width" numeric(10,2) NULL, "weight" numeric(10,2) NULL, "idling_fuel_usage" numeric(10,2) NULL, "exempt_from_tolls" boolean NOT NULL DEFAULT false, "color" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "equipment_types_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "equipment_types_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "equipment_types"
COMMENT ON COLUMN "equipment_types" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "equipment_types"
COMMENT ON COLUMN "equipment_types" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "equipment_types"
COMMENT ON COLUMN "equipment_types" ."version" IS 'The current version of this entity.';
-- Create "feasibility_tool_controls" table
CREATE TABLE "feasibility_tool_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "otp_operator" character varying NOT NULL DEFAULT 'Eq', "otp_value" double precision NOT NULL DEFAULT 100, "mpw_operator" character varying NOT NULL DEFAULT 'Eq', "mpw_value" double precision NOT NULL DEFAULT 100, "mpd_operator" character varying NOT NULL DEFAULT 'Eq', "mpd_value" double precision NOT NULL DEFAULT 100, "mpg_operator" character varying NOT NULL DEFAULT 'Eq', "mpg_value" double precision NOT NULL DEFAULT 100, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "feasibility_tool_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "feasibility_tool_controls_organizations_feasibility_tool_contro" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "feasibility_tool_controls_organization_id_key" to table: "feasibility_tool_controls"
CREATE UNIQUE INDEX "feasibility_tool_controls_organization_id_key" ON "feasibility_tool_controls" ("organization_id");
-- Create "users" table
CREATE TABLE "users" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "name" character varying NOT NULL, "username" character varying NOT NULL, "password" character varying NOT NULL, "email" character varying NOT NULL, "timezone" character varying(20) NOT NULL, "profile_pic_url" character varying NULL, "thumbnail_url" character varying NULL, "phone_number" character varying NULL, "is_admin" boolean NOT NULL DEFAULT false, "is_super_admin" boolean NOT NULL DEFAULT false, "last_login" timestamptz NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "users_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "users_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "user_username_email" to table: "users"
CREATE UNIQUE INDEX "user_username_email" ON "users" ("username", "email");
-- Set comment to column: "created_at" on table: "users"
COMMENT ON COLUMN "users" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "users"
COMMENT ON COLUMN "users" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "users"
COMMENT ON COLUMN "users" ."version" IS 'The current version of this entity.';
-- Create "fleet_codes" table
CREATE TABLE "fleet_codes" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'A', "code" character varying NOT NULL, "description" text NULL, "revenue_goal" numeric(10,2) NULL, "deadhead_goal" numeric(10,2) NULL, "mileage_goal" numeric(10,2) NULL, "color" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "manager_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "fleet_codes_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "fleet_codes_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "fleet_codes_users_manager" FOREIGN KEY ("manager_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL);
-- Set comment to column: "created_at" on table: "fleet_codes"
COMMENT ON COLUMN "fleet_codes" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "fleet_codes"
COMMENT ON COLUMN "fleet_codes" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "fleet_codes"
COMMENT ON COLUMN "fleet_codes" ."version" IS 'The current version of this entity.';
-- Create "formula_templates" table
CREATE TABLE "formula_templates" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying NOT NULL, "formula_text" text NOT NULL, "description" text NULL, "template_type" character varying NOT NULL DEFAULT 'General', "auto_apply" boolean NOT NULL DEFAULT false, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "customer_id" uuid NULL, "shipment_type_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "formula_templates_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "formula_templates_customers_customer" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "formula_templates_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "formula_templates_shipment_types_shipment_type" FOREIGN KEY ("shipment_type_id") REFERENCES "shipment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "formula_templates"
COMMENT ON COLUMN "formula_templates" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "formula_templates"
COMMENT ON COLUMN "formula_templates" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "formula_templates"
COMMENT ON COLUMN "formula_templates" ."version" IS 'The current version of this entity.';
-- Create "tags" table
CREATE TABLE "tags" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying(50) NOT NULL, "description" text NULL, "color" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "tags_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tags_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "tags"
COMMENT ON COLUMN "tags" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "tags"
COMMENT ON COLUMN "tags" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "tags"
COMMENT ON COLUMN "tags" ."version" IS 'The current version of this entity.';
-- Create "general_ledger_account_tags" table
CREATE TABLE "general_ledger_account_tags" ("general_ledger_account_id" uuid NOT NULL, "tag_id" uuid NOT NULL, PRIMARY KEY ("general_ledger_account_id", "tag_id"), CONSTRAINT "general_ledger_account_tags_general_ledger_account_id" FOREIGN KEY ("general_ledger_account_id") REFERENCES "general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "general_ledger_account_tags_tag_id" FOREIGN KEY ("tag_id") REFERENCES "tags" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create "google_apis" table
CREATE TABLE "google_apis" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "api_key" character varying NOT NULL, "mileage_unit" character varying NOT NULL DEFAULT 'Imperial', "add_customer_location" boolean NOT NULL DEFAULT false, "auto_geocode" boolean NOT NULL DEFAULT false, "add_location" boolean NOT NULL DEFAULT false, "traffic_model" character varying NOT NULL DEFAULT 'BestGuess', "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "google_apis_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "google_apis_organizations_google_api" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "google_apis_api_key_key" to table: "google_apis"
CREATE UNIQUE INDEX "google_apis_api_key_key" ON "google_apis" ("api_key");
-- Create index "google_apis_organization_id_key" to table: "google_apis"
CREATE UNIQUE INDEX "google_apis_organization_id_key" ON "google_apis" ("organization_id");
-- Create "hazardous_material_segregations" table
CREATE TABLE "hazardous_material_segregations" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "class_a" character varying(16) NOT NULL DEFAULT 'HazardClass1And1', "class_b" character varying(16) NOT NULL DEFAULT 'HazardClass1And1', "segregation_type" character varying(21) NOT NULL DEFAULT 'NotAllowed', "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "hazardous_material_segregations_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "hazardous_material_segregations_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "hazardousmaterialsegregation_class_a_class_b_organization_id" to table: "hazardous_material_segregations"
CREATE UNIQUE INDEX "hazardousmaterialsegregation_class_a_class_b_organization_id" ON "hazardous_material_segregations" ("class_a", "class_b", "organization_id");
-- Set comment to column: "created_at" on table: "hazardous_material_segregations"
COMMENT ON COLUMN "hazardous_material_segregations" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "hazardous_material_segregations"
COMMENT ON COLUMN "hazardous_material_segregations" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "hazardous_material_segregations"
COMMENT ON COLUMN "hazardous_material_segregations" ."version" IS 'The current version of this entity.';
-- Create "invoice_controls" table
CREATE TABLE "invoice_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "invoice_number_prefix" character varying(10) NOT NULL DEFAULT 'INV-', "credit_memo_number_prefix" character varying(10) NOT NULL DEFAULT 'CM-', "invoice_terms" text NULL, "invoice_footer" text NULL, "invoice_logo_url" character varying NULL, "invoice_date_format" character varying NOT NULL DEFAULT 'InvoiceDateFormatMDY', "invoice_due_after_days" smallint NOT NULL DEFAULT 30, "invoice_logo_width" smallint NOT NULL DEFAULT 100, "show_amount_due" boolean NOT NULL DEFAULT true, "attach_pdf" boolean NOT NULL DEFAULT true, "show_invoice_due_date" boolean NOT NULL DEFAULT true, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "invoice_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "invoice_controls_organizations_invoice_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "invoice_controls_organization_id_key" to table: "invoice_controls"
CREATE UNIQUE INDEX "invoice_controls_organization_id_key" ON "invoice_controls" ("organization_id");
-- Create "location_comments" table
CREATE TABLE "location_comments" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "comment" text NOT NULL, "location_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "user_id" uuid NOT NULL, "comment_type_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "location_comments_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "location_comments_comment_types_comment_type" FOREIGN KEY ("comment_type_id") REFERENCES "comment_types" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "location_comments_locations_comments" FOREIGN KEY ("location_id") REFERENCES "locations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "location_comments_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "location_comments_users_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "location_comments"
COMMENT ON COLUMN "location_comments" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "location_comments"
COMMENT ON COLUMN "location_comments" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "location_comments"
COMMENT ON COLUMN "location_comments" ."version" IS 'The current version of this entity.';
-- Create "location_contacts" table
CREATE TABLE "location_contacts" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying NOT NULL, "email_address" character varying NULL, "phone_number" character varying(15) NULL, "location_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "location_contacts_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "location_contacts_locations_contacts" FOREIGN KEY ("location_id") REFERENCES "locations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "location_contacts_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "location_contacts"
COMMENT ON COLUMN "location_contacts" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "location_contacts"
COMMENT ON COLUMN "location_contacts" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "location_contacts"
COMMENT ON COLUMN "location_contacts" ."version" IS 'The current version of this entity.';
-- Create "feature_flags" table
CREATE TABLE "feature_flags" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "name" character varying NOT NULL, "code" character varying(30) NOT NULL, "beta" boolean NOT NULL DEFAULT false, "description" text NOT NULL, "preview_picture_url" character varying NULL, PRIMARY KEY ("id"));
-- Create index "feature_flags_code_key" to table: "feature_flags"
CREATE UNIQUE INDEX "feature_flags_code_key" ON "feature_flags" ("code");
-- Set comment to table: "feature_flags"
COMMENT ON TABLE "feature_flags" IS 'Internal table for storing the feature flags available for Trenova';
-- Set comment to column: "created_at" on table: "feature_flags"
COMMENT ON COLUMN "feature_flags" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "feature_flags"
COMMENT ON COLUMN "feature_flags" ."updated_at" IS 'The last time that this entity was updated.';
-- Create "organization_feature_flags" table
CREATE TABLE "organization_feature_flags" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "is_enabled" boolean NOT NULL DEFAULT true, "feature_flag_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "organization_feature_flags_feature_flags_feature_flag" FOREIGN KEY ("feature_flag_id") REFERENCES "feature_flags" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "organization_feature_flags_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "organizationfeatureflag_organization_id_feature_flag_id" to table: "organization_feature_flags"
CREATE UNIQUE INDEX "organizationfeatureflag_organization_id_feature_flag_id" ON "organization_feature_flags" ("organization_id", "feature_flag_id");
-- Create "resources" table
CREATE TABLE "resources" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "type" character varying NOT NULL, "description" character varying NULL, PRIMARY KEY ("id"));
-- Create index "resources_type_key" to table: "resources"
CREATE UNIQUE INDEX "resources_type_key" ON "resources" ("type");
-- Create "permissions" table
CREATE TABLE "permissions" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "codename" character varying NOT NULL, "action" character varying NULL, "label" character varying NULL, "read_description" character varying NULL, "write_description" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "resource_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "permissions_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "permissions_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "permissions_resources_permissions" FOREIGN KEY ("resource_id") REFERENCES "resources" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "permissions"
COMMENT ON COLUMN "permissions" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "permissions"
COMMENT ON COLUMN "permissions" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "permissions"
COMMENT ON COLUMN "permissions" ."version" IS 'The current version of this entity.';
-- Create "qualifier_codes" table
CREATE TABLE "qualifier_codes" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "description" text NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "qualifier_codes_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "qualifier_codes_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "qualifier_codes"
COMMENT ON COLUMN "qualifier_codes" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "qualifier_codes"
COMMENT ON COLUMN "qualifier_codes" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "qualifier_codes"
COMMENT ON COLUMN "qualifier_codes" ."version" IS 'The current version of this entity.';
-- Create "reason_codes" table
CREATE TABLE "reason_codes" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying(1) NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "code_type" character varying NOT NULL, "description" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "reason_codes_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "reason_codes_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "reason_codes"
COMMENT ON COLUMN "reason_codes" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "reason_codes"
COMMENT ON COLUMN "reason_codes" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "reason_codes"
COMMENT ON COLUMN "reason_codes" ."version" IS 'The current version of this entity.';
-- Create "roles" table
CREATE TABLE "roles" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying NOT NULL, "description" character varying NULL, "color" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "roles_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "roles_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "roles"
COMMENT ON COLUMN "roles" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "roles"
COMMENT ON COLUMN "roles" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "roles"
COMMENT ON COLUMN "roles" ."version" IS 'The current version of this entity.';
-- Create "role_permissions" table
CREATE TABLE "role_permissions" ("role_id" uuid NOT NULL, "permission_id" uuid NOT NULL, PRIMARY KEY ("role_id", "permission_id"), CONSTRAINT "role_permissions_permission_id" FOREIGN KEY ("permission_id") REFERENCES "permissions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "role_permissions_role_id" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create "role_users" table
CREATE TABLE "role_users" ("role_id" uuid NOT NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("role_id", "user_id"), CONSTRAINT "role_users_role_id" FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "role_users_user_id" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create "route_controls" table
CREATE TABLE "route_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "distance_method" character varying(8) NOT NULL DEFAULT 'Trenova', "mileage_unit" character varying(14) NOT NULL DEFAULT 'UnitsMetric', "generate_routes" boolean NOT NULL DEFAULT false, "organization_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "route_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "route_controls_organizations_route_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "route_controls_organization_id_key" to table: "route_controls"
CREATE UNIQUE INDEX "route_controls_organization_id_key" ON "route_controls" ("organization_id");
-- Create "service_types" table
CREATE TABLE "service_types" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "description" text NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "service_types_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "service_types_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "service_types"
COMMENT ON COLUMN "service_types" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "service_types"
COMMENT ON COLUMN "service_types" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "service_types"
COMMENT ON COLUMN "service_types" ."version" IS 'The current version of this entity.';
-- Create "shipments" table
CREATE TABLE "shipments" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "pro_number" character varying(20) NOT NULL, "status" character varying NOT NULL DEFAULT 'New', "origin_address_line" character varying NULL, "origin_appointment_start" timestamptz NULL, "origin_appointment_end" timestamptz NULL, "destination_address_line" character varying NULL, "destination_appointment_start" timestamptz NULL, "destination_appointment_end" timestamptz NULL, "rating_unit" bigint NOT NULL DEFAULT 1, "mileage" double precision NULL, "other_charge_amount" numeric(19,4) NULL, "freight_charge_amount" numeric(19,4) NULL, "rating_method" character varying NOT NULL DEFAULT 'FlatRate', "pieces" numeric(10,2) NULL, "weight" numeric(10,2) NULL, "ready_to_bill" boolean NOT NULL DEFAULT false, "bill_date" date NULL, "ship_date" date NULL, "billed" boolean NOT NULL DEFAULT false, "transferred_to_billing" boolean NOT NULL DEFAULT false, "transferred_to_billing_date" date NULL, "total_charge_amount" numeric(19,4) NULL, "temperature_min" bigint NULL, "temperature_max" bigint NULL, "bill_of_lading_number" character varying NULL, "consignee_reference_number" character varying NULL, "comment" text NULL, "voided_comment" character varying(100) NULL, "auto_rated" boolean NOT NULL DEFAULT false, "current_suffix" character varying(2) NULL, "entry_method" character varying NOT NULL DEFAULT 'Manual', "is_hazardous" boolean NOT NULL DEFAULT false, "customer_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "shipment_type_id" uuid NOT NULL, "service_type_id" uuid NULL, "revenue_code_id" uuid NULL, "origin_location_id" uuid NULL, "destination_location_id" uuid NULL, "trailer_type_id" uuid NULL, "tractor_type_id" uuid NULL, "created_by" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "shipments_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_customers_shipments" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT, CONSTRAINT "shipments_equipment_types_tractor_type" FOREIGN KEY ("tractor_type_id") REFERENCES "equipment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_equipment_types_trailer_type" FOREIGN KEY ("trailer_type_id") REFERENCES "equipment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_locations_destination_location" FOREIGN KEY ("destination_location_id") REFERENCES "locations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_locations_origin_location" FOREIGN KEY ("origin_location_id") REFERENCES "locations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_service_types_revenue_code" FOREIGN KEY ("revenue_code_id") REFERENCES "service_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_service_types_service_type" FOREIGN KEY ("service_type_id") REFERENCES "service_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_shipment_types_shipment_type" FOREIGN KEY ("shipment_type_id") REFERENCES "shipment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipments_users_shipments" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL);
-- Create index "shipment_bill_date_organization_id" to table: "shipments"
CREATE INDEX "shipment_bill_date_organization_id" ON "shipments" ("bill_date", "organization_id");
-- Create index "shipment_bill_of_lading_number_organization_id" to table: "shipments"
CREATE INDEX "shipment_bill_of_lading_number_organization_id" ON "shipments" ("bill_of_lading_number", "organization_id");
-- Create index "shipment_ship_date_organization_id" to table: "shipments"
CREATE INDEX "shipment_ship_date_organization_id" ON "shipments" ("ship_date", "organization_id");
-- Create index "shipment_status" to table: "shipments"
CREATE INDEX "shipment_status" ON "shipments" ("status");
-- Set comment to table: "shipments"
COMMENT ON TABLE "shipments" IS 'Shipment holds the schema definition for the Shipment entity.';
-- Set comment to column: "created_at" on table: "shipments"
COMMENT ON COLUMN "shipments" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipments"
COMMENT ON COLUMN "shipments" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipments"
COMMENT ON COLUMN "shipments" ."version" IS 'The current version of this entity.';
-- Set comment to column: "rating_unit" on table: "shipments"
COMMENT ON COLUMN "shipments" ."rating_unit" IS 'The rating unit for the shipment.';
-- Set comment to column: "voided_comment" on table: "shipments"
COMMENT ON COLUMN "shipments" ."voided_comment" IS 'The comment for voiding the shipment.';
-- Set comment to column: "auto_rated" on table: "shipments"
COMMENT ON COLUMN "shipments" ."auto_rated" IS 'Indicates if the shipment was auto rated.';
-- Set comment to column: "is_hazardous" on table: "shipments"
COMMENT ON COLUMN "shipments" ."is_hazardous" IS 'Indicates if the shipment is hazardous.';
-- Create "shipment_charges" table
CREATE TABLE "shipment_charges" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "description" text NOT NULL, "charge_amount" numeric(19,4) NOT NULL, "units" bigint NOT NULL, "sub_total" numeric(19,4) NOT NULL, "accessorial_charge_id" uuid NOT NULL, "shipment_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "created_by" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_charges_accessorial_charges_shipment_charges" FOREIGN KEY ("accessorial_charge_id") REFERENCES "accessorial_charges" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "shipment_charges_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_charges_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_charges_shipments_shipment_charges" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_charges_users_shipment_charges" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "shipment_charges"
COMMENT ON COLUMN "shipment_charges" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipment_charges"
COMMENT ON COLUMN "shipment_charges" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipment_charges"
COMMENT ON COLUMN "shipment_charges" ."version" IS 'The current version of this entity.';
-- Create "shipment_comments" table
CREATE TABLE "shipment_comments" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "comment" text NOT NULL, "comment_type_id" uuid NOT NULL, "shipment_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "created_by" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_comments_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_comments_comment_types_shipment_comments" FOREIGN KEY ("comment_type_id") REFERENCES "comment_types" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "shipment_comments_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_comments_shipments_shipment_comments" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_comments_users_shipment_comments" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "shipment_comments"
COMMENT ON COLUMN "shipment_comments" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipment_comments"
COMMENT ON COLUMN "shipment_comments" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipment_comments"
COMMENT ON COLUMN "shipment_comments" ."version" IS 'The current version of this entity.';
-- Create "shipment_commodities" table
CREATE TABLE "shipment_commodities" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "commodity_id" uuid NOT NULL, "hazardous_material_id" uuid NOT NULL, "sub_total" numeric(10,2) NOT NULL, "placard_needed" boolean NOT NULL DEFAULT false, "shipment_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_commodities_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_commodities_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_commodities_shipments_shipment_commodities" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "shipment_commodities"
COMMENT ON COLUMN "shipment_commodities" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipment_commodities"
COMMENT ON COLUMN "shipment_commodities" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipment_commodities"
COMMENT ON COLUMN "shipment_commodities" ."version" IS 'The current version of this entity.';
-- Create "shipment_controls" table
CREATE TABLE "shipment_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "auto_rate_shipment" boolean NOT NULL DEFAULT true, "calculate_distance" boolean NOT NULL DEFAULT true, "enforce_rev_code" boolean NOT NULL DEFAULT false, "enforce_voided_comm" boolean NOT NULL DEFAULT false, "generate_routes" boolean NOT NULL DEFAULT false, "enforce_commodity" boolean NOT NULL DEFAULT false, "auto_sequence_stops" boolean NOT NULL DEFAULT true, "auto_shipment_total" boolean NOT NULL DEFAULT true, "enforce_origin_destination" boolean NOT NULL DEFAULT false, "check_for_duplicate_bol" boolean NOT NULL DEFAULT false, "send_placard_info" boolean NOT NULL DEFAULT false, "enforce_hazmat_seg_rules" boolean NOT NULL DEFAULT true, "organization_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_controls_organizations_shipment_control" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "shipment_controls_organization_id_key" to table: "shipment_controls"
CREATE UNIQUE INDEX "shipment_controls_organization_id_key" ON "shipment_controls" ("organization_id");
-- Create "shipment_documentations" table
CREATE TABLE "shipment_documentations" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "document_url" character varying NOT NULL, "document_classification_id" uuid NOT NULL, "shipment_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_documentations_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_documentations_document_classifications_shipment_docum" FOREIGN KEY ("document_classification_id") REFERENCES "document_classifications" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "shipment_documentations_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_documentations_shipments_shipment_documentation" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "shipment_documentations"
COMMENT ON COLUMN "shipment_documentations" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipment_documentations"
COMMENT ON COLUMN "shipment_documentations" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipment_documentations"
COMMENT ON COLUMN "shipment_documentations" ."version" IS 'The current version of this entity.';
-- Create "workers" table
CREATE TABLE "workers" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'A', "code" character varying(10) NOT NULL, "profile_picture_url" character varying NULL, "worker_type" character varying NOT NULL DEFAULT 'Employee', "first_name" character varying NOT NULL, "last_name" character varying NOT NULL, "address_line_1" character varying(150) NULL, "address_line_2" character varying(150) NULL, "city" character varying(150) NULL, "postal_code" character varying(10) NULL, "external_id" character varying NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "state_id" uuid NULL, "fleet_code_id" uuid NULL, "manager_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "workers_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_fleet_codes_fleet_code" FOREIGN KEY ("fleet_code_id") REFERENCES "fleet_codes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "workers_users_manager" FOREIGN KEY ("manager_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "worker_first_name_last_name" to table: "workers"
CREATE INDEX "worker_first_name_last_name" ON "workers" ("first_name", "last_name");
-- Set comment to column: "created_at" on table: "workers"
COMMENT ON COLUMN "workers" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "workers"
COMMENT ON COLUMN "workers" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "workers"
COMMENT ON COLUMN "workers" ."version" IS 'The current version of this entity.';
-- Set comment to column: "external_id" on table: "workers"
COMMENT ON COLUMN "workers" ."external_id" IS 'External ID usually from HOS integration.';
-- Create "tractors" table
CREATE TABLE "tractors" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "code" character varying(50) NOT NULL, "status" character varying(13) NOT NULL DEFAULT 'Available', "license_plate_number" character varying(50) NULL, "vin" character varying(17) NULL, "model" character varying(50) NULL, "year" smallint NULL, "leased" boolean NOT NULL DEFAULT false, "leased_date" date NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "equipment_type_id" uuid NULL, "equipment_manufacturer_id" uuid NULL, "state_id" uuid NULL, "fleet_code_id" uuid NOT NULL, "primary_worker_id" uuid NOT NULL, "secondary_worker_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "tractors_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_equipment_manufactuers_equipment_manufacturer" FOREIGN KEY ("equipment_manufacturer_id") REFERENCES "equipment_manufactuers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_equipment_types_equipment_type" FOREIGN KEY ("equipment_type_id") REFERENCES "equipment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_fleet_codes_fleet_code" FOREIGN KEY ("fleet_code_id") REFERENCES "fleet_codes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_workers_primary_tractor" FOREIGN KEY ("primary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "tractors_workers_secondary_tractor" FOREIGN KEY ("secondary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "tractors_primary_worker_id_key" to table: "tractors"
CREATE UNIQUE INDEX "tractors_primary_worker_id_key" ON "tractors" ("primary_worker_id");
-- Create index "tractors_secondary_worker_id_key" to table: "tractors"
CREATE UNIQUE INDEX "tractors_secondary_worker_id_key" ON "tractors" ("secondary_worker_id");
-- Set comment to column: "created_at" on table: "tractors"
COMMENT ON COLUMN "tractors" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "tractors"
COMMENT ON COLUMN "tractors" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "tractors"
COMMENT ON COLUMN "tractors" ."version" IS 'The current version of this entity.';
-- Set comment to column: "code" on table: "tractors"
COMMENT ON COLUMN "tractors" ."code" IS 'The unique code assigned to each tractor for identification purposes.';
-- Set comment to column: "status" on table: "tractors"
COMMENT ON COLUMN "tractors" ."status" IS 'The operational status of the tractor, indicating availability, maintenance, or other conditions.';
-- Set comment to column: "license_plate_number" on table: "tractors"
COMMENT ON COLUMN "tractors" ."license_plate_number" IS 'The license plate number of the tractor, used for legal identification on roads.';
-- Set comment to column: "vin" on table: "tractors"
COMMENT ON COLUMN "tractors" ."vin" IS 'The Vehicle Identification Number, a unique code used to identify individual motor vehicles.';
-- Set comment to column: "model" on table: "tractors"
COMMENT ON COLUMN "tractors" ."model" IS 'The model of the tractor, which indicates the design and technical specifications.';
-- Set comment to column: "year" on table: "tractors"
COMMENT ON COLUMN "tractors" ."year" IS 'The year the tractor was manufactured, reflecting its age and potentially its technology level.';
-- Set comment to column: "leased" on table: "tractors"
COMMENT ON COLUMN "tractors" ."leased" IS 'Indicates whether the tractor is currently leased or owned outright.';
-- Set comment to column: "leased_date" on table: "tractors"
COMMENT ON COLUMN "tractors" ."leased_date" IS 'The date on which the tractor was leased, if applicable.';
-- Set comment to column: "equipment_type_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."equipment_type_id" IS 'Identifier for the type of equipment the tractor is classified under.';
-- Set comment to column: "equipment_manufacturer_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."equipment_manufacturer_id" IS 'The UUID of the manufacturer of the tractor''s equipment, linking to specific company details.';
-- Set comment to column: "state_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."state_id" IS 'A UUID representing the state in which the tractor is registered, for jurisdiction purposes.';
-- Set comment to column: "fleet_code_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."fleet_code_id" IS 'A UUID linking the tractor to a specific fleet within an organization.';
-- Set comment to column: "primary_worker_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."primary_worker_id" IS 'The primary worker assigned to operate the tractor, identified by UUID.';
-- Set comment to column: "secondary_worker_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."secondary_worker_id" IS 'An optional secondary worker who can also operate the tractor, identified by UUID.';
-- Create "shipment_moves" table
CREATE TABLE "shipment_moves" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "reference_number" character varying(10) NOT NULL, "status" character varying NOT NULL DEFAULT 'New', "is_loaded" boolean NOT NULL DEFAULT false, "shipment_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "tractor_id" uuid NULL, "trailer_id" uuid NULL, "primary_worker_id" uuid NULL, "secondary_worker_id" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_moves_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_moves_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_moves_shipments_shipment_moves" FOREIGN KEY ("shipment_id") REFERENCES "shipments" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "shipment_moves_tractors_tractor" FOREIGN KEY ("tractor_id") REFERENCES "tractors" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_moves_tractors_trailer" FOREIGN KEY ("trailer_id") REFERENCES "tractors" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_moves_workers_primary_worker" FOREIGN KEY ("primary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_moves_workers_secondary_worker" FOREIGN KEY ("secondary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Create index "shipment_moves_reference_number_key" to table: "shipment_moves"
CREATE UNIQUE INDEX "shipment_moves_reference_number_key" ON "shipment_moves" ("reference_number");
-- Set comment to column: "created_at" on table: "shipment_moves"
COMMENT ON COLUMN "shipment_moves" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipment_moves"
COMMENT ON COLUMN "shipment_moves" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipment_moves"
COMMENT ON COLUMN "shipment_moves" ."version" IS 'The current version of this entity.';
-- Create "shipment_routes" table
CREATE TABLE "shipment_routes" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "mileage" double precision NOT NULL, "duration" bigint NULL, "distance_method" character varying(50) NULL, "auto_generated" boolean NOT NULL DEFAULT false, "origin_location_id" uuid NOT NULL, "destination_location_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "shipment_routes_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_routes_locations_destination_route_locations" FOREIGN KEY ("destination_location_id") REFERENCES "locations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_routes_locations_origin_route_locations" FOREIGN KEY ("origin_location_id") REFERENCES "locations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "shipment_routes_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "shipment_routes"
COMMENT ON COLUMN "shipment_routes" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "shipment_routes"
COMMENT ON COLUMN "shipment_routes" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "shipment_routes"
COMMENT ON COLUMN "shipment_routes" ."version" IS 'The current version of this entity.';
-- Create "stops" table
CREATE TABLE "stops" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'New', "stop_type" character varying NOT NULL, "sequence" bigint NOT NULL DEFAULT 1, "location_id" uuid NULL, "pieces" numeric(10,2) NULL, "weight" numeric(10,2) NULL, "address_line" character varying NULL, "appointment_start" timestamptz NULL, "appointment_end" timestamptz NULL, "arrival_time" timestamptz NULL, "departure_time" timestamptz NULL, "shipment_move_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "stops_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "stops_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "stops_shipment_moves_move_stops" FOREIGN KEY ("shipment_move_id") REFERENCES "shipment_moves" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "stops"
COMMENT ON COLUMN "stops" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "stops"
COMMENT ON COLUMN "stops" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "stops"
COMMENT ON COLUMN "stops" ."version" IS 'The current version of this entity.';
-- Set comment to column: "sequence" on table: "stops"
COMMENT ON COLUMN "stops" ."sequence" IS 'Current sequence of the stop within the movement.';
-- Create "table_change_alerts" table
CREATE TABLE "table_change_alerts" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "status" character varying NOT NULL DEFAULT 'A', "name" character varying(50) NOT NULL, "database_action" character varying(6) NOT NULL, "source" character varying NOT NULL, "table_name" character varying NULL, "topic_name" character varying NULL, "description" text NULL, "custom_subject" character varying NULL, "function_name" character varying(50) NULL, "trigger_name" character varying(50) NULL, "listener_name" character varying(50) NULL, "email_recipients" text NULL, "effective_date" date NULL, "expiration_date" date NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "table_change_alerts_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "table_change_alerts_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "table_change_alerts"
COMMENT ON COLUMN "table_change_alerts" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "table_change_alerts"
COMMENT ON COLUMN "table_change_alerts" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "table_change_alerts"
COMMENT ON COLUMN "table_change_alerts" ."version" IS 'The current version of this entity.';
-- Create "trailers" table
CREATE TABLE "trailers" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "code" character varying(50) NOT NULL, "status" character varying(13) NOT NULL DEFAULT 'Available', "vin" character varying(17) NULL, "model" character varying(50) NULL, "year" smallint NULL, "license_plate_number" character varying(50) NULL, "last_inspection_date" date NULL, "registration_number" character varying NULL, "registration_expiration_date" date NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "equipment_type_id" uuid NOT NULL, "equipment_manufacturer_id" uuid NULL, "state_id" uuid NULL, "registration_state_id" uuid NULL, "fleet_code_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "trailers_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "trailers_equipment_manufactuers_equipment_manufacturer" FOREIGN KEY ("equipment_manufacturer_id") REFERENCES "equipment_manufactuers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "trailers_equipment_types_equipment_type" FOREIGN KEY ("equipment_type_id") REFERENCES "equipment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "trailers_fleet_codes_fleet_code" FOREIGN KEY ("fleet_code_id") REFERENCES "fleet_codes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "trailers_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "trailers_us_states_registration_state" FOREIGN KEY ("registration_state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "trailers_us_states_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE);
-- Set comment to column: "created_at" on table: "trailers"
COMMENT ON COLUMN "trailers" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "trailers"
COMMENT ON COLUMN "trailers" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "trailers"
COMMENT ON COLUMN "trailers" ."version" IS 'The current version of this entity.';
-- Create "user_favorites" table
CREATE TABLE "user_favorites" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "page_link" character varying NOT NULL, "user_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "user_favorites_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "user_favorites_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "user_favorites_users_user_favorites" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "user_favorites_page_link_key" to table: "user_favorites"
CREATE UNIQUE INDEX "user_favorites_page_link_key" ON "user_favorites" ("page_link");
-- Set comment to column: "created_at" on table: "user_favorites"
COMMENT ON COLUMN "user_favorites" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "user_favorites"
COMMENT ON COLUMN "user_favorites" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "user_favorites"
COMMENT ON COLUMN "user_favorites" ."version" IS 'The current version of this entity.';
-- Create "user_notifications" table
CREATE TABLE "user_notifications" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "is_read" boolean NOT NULL DEFAULT false, "title" character varying NOT NULL, "description" text NOT NULL, "action_url" character varying NULL, "user_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "user_notifications_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "user_notifications_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "user_notifications_users_user_notifications" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "user_notifications"
COMMENT ON COLUMN "user_notifications" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "user_notifications"
COMMENT ON COLUMN "user_notifications" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "user_notifications"
COMMENT ON COLUMN "user_notifications" ."version" IS 'The current version of this entity.';
-- Set comment to column: "action_url" on table: "user_notifications"
COMMENT ON COLUMN "user_notifications" ."action_url" IS 'URL to redirect the user to when the notification is clicked.';
-- Create "user_reports" table
CREATE TABLE "user_reports" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "report_url" character varying NOT NULL, "user_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "user_reports_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "user_reports_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "user_reports_users_reports" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "user_reports"
COMMENT ON COLUMN "user_reports" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "user_reports"
COMMENT ON COLUMN "user_reports" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "user_reports"
COMMENT ON COLUMN "user_reports" ."version" IS 'The current version of this entity.';
-- Create "worker_comments" table
CREATE TABLE "worker_comments" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "comment" text NOT NULL, "worker_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "comment_type_id" uuid NOT NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "worker_comments_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_comments_comment_types_comment_type" FOREIGN KEY ("comment_type_id") REFERENCES "comment_types" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_comments_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_comments_users_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT "worker_comments_workers_worker_comments" FOREIGN KEY ("worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "worker_comments"
COMMENT ON COLUMN "worker_comments" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "worker_comments"
COMMENT ON COLUMN "worker_comments" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "worker_comments"
COMMENT ON COLUMN "worker_comments" ."version" IS 'The current version of this entity.';
-- Create "worker_contacts" table
CREATE TABLE "worker_contacts" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "name" character varying NOT NULL, "email" character varying NOT NULL, "phone" character varying NOT NULL, "relationship" character varying NULL, "is_primary" boolean NOT NULL DEFAULT false, "worker_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "worker_contacts_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_contacts_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_contacts_workers_worker_contacts" FOREIGN KEY ("worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Set comment to column: "created_at" on table: "worker_contacts"
COMMENT ON COLUMN "worker_contacts" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "worker_contacts"
COMMENT ON COLUMN "worker_contacts" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "worker_contacts"
COMMENT ON COLUMN "worker_contacts" ."version" IS 'The current version of this entity.';
-- Create "worker_profiles" table
CREATE TABLE "worker_profiles" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "version" bigint NOT NULL DEFAULT 1, "race" character varying NULL, "sex" character varying NULL, "date_of_birth" date NULL, "license_number" character varying NOT NULL, "license_expiration_date" date NULL, "endorsements" character varying NULL DEFAULT 'None', "hazmat_expiration_date" date NULL, "hire_date" date NULL, "termination_date" date NULL, "physical_due_date" date NULL, "medical_cert_date" date NULL, "mvr_due_date" date NULL, "worker_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, "organization_id" uuid NOT NULL, "license_state_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "worker_profiles_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_profiles_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_profiles_us_states_state" FOREIGN KEY ("license_state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT "worker_profiles_workers_worker_profile" FOREIGN KEY ("worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "worker_profiles_worker_id_key" to table: "worker_profiles"
CREATE UNIQUE INDEX "worker_profiles_worker_id_key" ON "worker_profiles" ("worker_id");
-- Set comment to column: "created_at" on table: "worker_profiles"
COMMENT ON COLUMN "worker_profiles" ."created_at" IS 'The time that this entity was created.';
-- Set comment to column: "updated_at" on table: "worker_profiles"
COMMENT ON COLUMN "worker_profiles" ."updated_at" IS 'The last time that this entity was updated.';
-- Set comment to column: "version" on table: "worker_profiles"
COMMENT ON COLUMN "worker_profiles" ."version" IS 'The current version of this entity.';
