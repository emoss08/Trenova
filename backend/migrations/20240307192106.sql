-- Create "business_units" table
CREATE TABLE
  "public"."business_units" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "status" "public"."status_type" NOT NULL DEFAULT 'A',
    "name" character varying(255) NOT NULL,
    "entity_key" character varying(10) NOT NULL,
    "contact_name" character varying(255) NULL,
    "contact_email" text NOT NULL,
    "paid_until" timestamptz NULL,
    "phone_number" character varying(15) NULL,
    "address" text NULL,
    "city" character varying(255) NULL,
    "state" character varying(2) NULL,
    "country" character varying(2) NULL,
    "postal_code" character varying(10) NULL,
    "parent_id" uuid NULL,
    "settings" jsonb NULL,
    "tax_id" character varying(20) NULL,
    "subscription_plan" text NOT NULL,
    "description" text NULL,
    "free_trial" boolean NOT NULL DEFAULT false,
    "legal_name" text NOT NULL,
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_business_units_parent" FOREIGN KEY ("parent_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create index "uni_business_units_entity_key" to table: "business_units"
CREATE UNIQUE INDEX "uni_business_units_entity_key" ON "public"."business_units" (LOWER("entity_key"));

-- Create index "uni_business_units_name" to table: "business_units"
CREATE UNIQUE INDEX "uni_business_units_name" ON "public"."business_units" (LOWER("name"));

-- Create index "idx_business_unit_parent_id" to table: "business_units"
CREATE INDEX "idx_business_unit_parent_id" ON "business_units" ("parent_id");

-- Create "organizations" table
CREATE TABLE
  "public"."organizations" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "name" character varying(255) NOT NULL,
    "scac_code" character varying(4) NOT NULL,
    "org_type" "public"."org_type" NOT NULL,
    "dot_number" character varying(12) NOT NULL,
    "logo_url" character varying(255) NULL,
    "timezone" "public"."timezone_type" NOT NULL DEFAULT 'America/Los_Angeles',
    "business_unit_id" uuid NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_organizations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create index "uni_organizations_dot_number" to table: "organizations"
CREATE UNIQUE INDEX "uni_organizations_dot_number" ON "public"."organizations" ("dot_number");

-- Create index "uni_organizations_scac_code" to table: "organizations"
CREATE UNIQUE INDEX "uni_organizations_scac_code" ON "public"."organizations" ("scac_code");

-- Create unique index for name and business_unit_id with LOWER applied, this will ensure that the name is unique within the business unit
CREATE UNIQUE INDEX "uni_organizations_name" ON "organizations" (LOWER("name"), "business_unit_id");

-- Create index "idx_organizations_business_unit" to table: "organizations"
CREATE INDEX "idx_organizations_business_unit" ON "organizations" ("business_unit_id");

-- Create "general_ledger_accounts" table
CREATE TABLE
  "public"."general_ledger_accounts" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "status" "public"."status_type" NOT NULL DEFAULT 'A',
    "account_number" character varying(7) NOT NULL,
    "account_type" "public"."ac_account_type" NOT NULL,
    "cash_flow_type" "public"."ac_cash_flow_type" NULL,
    "account_sub_type" "public"."ac_account_sub_type" NULL,
    "account_class" "public"."ac_account_classification" NULL,
    "balance" numeric(20, 2) NULL,
    "is_reconciled" boolean NOT NULL DEFAULT false,
    "date_opened" date NULL,
    "date_closed" date NULL,
    "notes" text NULL,
    "is_tax_relevant" boolean NOT NULL DEFAULT false,
    "interest_rate" numeric(5, 2) NULL,
    "test" text NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_general_ledger_accounts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_general_ledger_accounts_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create index "idx_general_ledger_account_organization" to table: "general_ledger_accounts"
CREATE UNIQUE INDEX "uni_general_ledger_account_organization" ON "general_ledger_accounts" (LOWER("account_number"), "organization_id");

-- Create index "idx_general_ledger_account_organization" to table: "general_ledger_accounts"
CREATE INDEX "idx_general_ledger_account_organization" ON "general_ledger_accounts" ("organization_id");

-- Create index "idx_general_ledger_account_business_unit" to table: "general_ledger_accounts"
CREATE INDEX "idx_general_ledger_account_business_unit" ON "general_ledger_accounts" ("business_unit_id");

-- Create index "idx_general_ledger_account_account_number" to table: "general_ledger_accounts"
CREATE INDEX "idx_general_ledger_account_account_number" ON "general_ledger_accounts" (LOWER("account_number"));

-- Create "accounting_controls" table
CREATE TABLE
  "public"."accounting_controls" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "business_unit_id" uuid NOT NULL,
    "organization_id" uuid NOT NULL,
    "auto_create_journal_entries" boolean NOT NULL DEFAULT false,
    "journal_entry_criteria" character varying(50) NULL DEFAULT 'ON_SHIPMENT_BILL',
    "restrict_manual_journal_entries" boolean NOT NULL DEFAULT false,
    "require_journal_entry_apporval" boolean NOT NULL DEFAULT false,
    "enable_rec_notifications" boolean NOT NULL DEFAULT true,
    "rec_threshold" bigint NOT NULL DEFAULT 50,
    "rec_threshold_action" "public"."ac_threshold_action_type" NOT NULL DEFAULT 'HALT',
    "default_revenue_account_id" uuid NULL,
    "default_expense_account_id" uuid NULL,
    "halt_on_pending_rec" boolean NOT NULL DEFAULT false,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_accounting_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_accounting_controls_default_expense_account" FOREIGN KEY ("default_expense_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_accounting_controls_default_revenue_account" FOREIGN KEY ("default_revenue_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_organizations_accounting_control" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "accounting_controls_organization_id_key" UNIQUE ("organization_id"), -- This is a unique constraint to ensure that there is only one accounting control per organization
    CONSTRAINT "accounting_control_reconciliation_threshold_check" CHECK ("rec_threshold" >= 0) -- This is a check constraint to ensure that the reconciliation threshold is greater than or equal to 0
  );

-- Create index "idx_general_ledger_account_organization" to table: "general_ledger_accounts"
CREATE INDEX "idx_accounting_controls_organization_id" ON "accounting_controls" ("organization_id");

-- Create index "idx_accounting_controls_business_unit" to table: "accounting_controls"
CREATE INDEX "idx_accounting_controls_default_revenue_account_id" ON "accounting_controls" ("default_revenue_account_id");

-- Create index "idx_accounting_controls_default_expense_account_id" to table: "accounting_controls"
CREATE INDEX "idx_accounting_controls_default_expense_account_id" ON "accounting_controls" ("default_expense_account_id");

-- Create "division_codes" table
CREATE TABLE
  "public"."division_codes" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "status" "public"."status_type" NOT NULL DEFAULT 'A',
    "code" character varying(4) NOT NULL,
    "description" character varying(100) NOT NULL,
    "cash_account_id" uuid NULL,
    "ap_account_id" uuid NULL,
    "expense_account_id" uuid NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_division_codes_ap_account" FOREIGN KEY ("ap_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT "fk_division_codes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_division_codes_cash_account" FOREIGN KEY ("cash_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT "fk_division_codes_expense_account" FOREIGN KEY ("expense_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT "fk_division_codes_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create unique index for code and organization_id with LOWER applied
CREATE UNIQUE INDEX "uni_division_code_organization_code" ON "division_codes" (LOWER("code"), "organization_id");

-- Create index "idx_division_code_organization" to table: "division_codes"
CREATE INDEX "idx_division_code_organization" ON "division_codes" ("organization_id");

-- Create index "idx_division_code_business_unit" to table: "division_codes"
CREATE INDEX "idx_division_code_business_unit" ON "division_codes" ("business_unit_id");

-- Create index "idx_division_code_cash_account_id" to table: "division_codes"
CREATE INDEX "idx_division_code_cash_account_id" ON "division_codes" ("cash_account_id");

-- Create index "idx_division_code_ap_account_id" to table: "division_codes"
CREATE INDEX "idx_division_code_ap_account_id" ON "division_codes" ("ap_account_id");

-- Create index "idx_division_code_expense_account_id" to table: "division_codes"
CREATE INDEX "idx_division_code_expense_account_id" ON "division_codes" ("expense_account_id");

-- Create index "idx_division_code_code" to table: "division_codes"
CREATE INDEX "idx_division_code_code" ON "division_codes" (LOWER("code"));

-- Create "email_profiles" table
CREATE TABLE
  "public"."email_profiles" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "name" character varying(255) NOT NULL,
    "email" character varying(255) NOT NULL,
    "protocol" "public"."email_protocol_type" NOT NULL,
    "host" character varying(255) NOT NULL,
    "port" integer NOT NULL,
    "username" character varying(255) NOT NULL,
    "password" character varying(255) NOT NULL,
    "default_profile" boolean NOT NULL DEFAULT false,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_email_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_email_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create "tags" table
CREATE TABLE
  "public"."tags" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "name" character varying(50) NOT NULL,
    "description" text NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_tags_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_tags_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create unique index for name and organization_id with LOWER applied
CREATE UNIQUE INDEX tag_organization_uqx ON tags (LOWER(name), organization_id);

-- Create "general_ledger_account_tags" table
CREATE TABLE
  "public"."general_ledger_account_tags" (
    "general_ledger_account_id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "tag_id" uuid NOT NULL DEFAULT gen_random_uuid (),
    PRIMARY KEY ("general_ledger_account_id", "tag_id"),
    CONSTRAINT "fk_general_ledger_account_tags_general_ledger_account" FOREIGN KEY ("general_ledger_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_general_ledger_account_tags_tag" FOREIGN KEY ("tag_id") REFERENCES "public"."tags" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create index "idx_general_ledger_account_tags_general_ledger_account_id" to table: "general_ledger_account_tags"
CREATE INDEX "idx_general_ledger_account_tags_general_ledger_account_id" ON "general_ledger_account_tags" ("general_ledger_account_id");

-- Create index "idx_general_ledger_account_tags_tag_id" to table: "general_ledger_account_tags"
CREATE INDEX "idx_general_ledger_account_tags_tag_id" ON "general_ledger_account_tags" ("tag_id");

-- Create "job_titles" table
CREATE TABLE
  "public"."job_titles" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "status" "public"."status_type" NOT NULL DEFAULT 'A',
    "name" character varying(100) NOT NULL,
    "description" character varying(100) NULL,
    "job_function" "public"."job_function_type" NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_job_titles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_job_titles_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create unique index for name and organization_id with LOWER applied
CREATE UNIQUE INDEX "uni_job_title_organization" ON "job_titles" (LOWER("name"), "organization_id");

-- Create index "idx_job_title_organization" to table: "job_titles"
CREATE INDEX "idx_job_title_organization" ON "job_titles" ("organization_id");

-- Create index "idx_job_title_business_unit" to table: "job_titles"
CREATE INDEX "idx_job_title_business_unit" ON "job_titles" ("business_unit_id");

-- Create index "idx_job_title_name" to table: "job_titles"
CREATE INDEX "idx_job_title_name" ON "job_titles" (LOWER("name"));

-- Create "revenue_codes" table
CREATE TABLE
  "public"."revenue_codes" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "code" character varying(4) NOT NULL,
    "description" character varying(100) NOT NULL,
    "expense_account_id" uuid NULL,
    "revenue_account_id" uuid NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_revenue_codes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_revenue_codes_expense_account" FOREIGN KEY ("expense_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT "fk_revenue_codes_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_revenue_codes_revenue_account" FOREIGN KEY ("revenue_account_id") REFERENCES "public"."general_ledger_accounts" ("id") ON UPDATE CASCADE ON DELETE SET NULL
  );

-- Create unique index for code and organization_id with LOWER applied
CREATE UNIQUE INDEX "uni_revenue_code_organization" ON "revenue_codes" (LOWER("code"), "organization_id");

-- Create index "idx_revenue_code_organization" to table: "revenue_codes"
CREATE INDEX "idx_revenue_code_organization" ON "revenue_codes" ("organization_id");

-- Create index "idx_revenue_code_business_unit" to table: "revenue_codes"
CREATE INDEX "idx_revenue_code_business_unit" ON "revenue_codes" ("business_unit_id");

-- Create index "idx_revenue_code_expense_account_id" to table: "revenue_codes"
CREATE INDEX "idx_revenue_code_expense_account_id" ON "revenue_codes" ("expense_account_id");

-- Create index "idx_revenue_code_revenue_account_id" to table: "revenue_codes"
CREATE INDEX "idx_revenue_code_revenue_account_id" ON "revenue_codes" ("revenue_account_id");

-- Create index "idx_revenue_code_code" to table: "revenue_codes"
CREATE INDEX "idx_revenue_code_code" ON "revenue_codes" (LOWER("code"));

-- Create "table_change_alerts" table
CREATE TABLE
  "public"."table_change_alerts" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "status" "public"."status_type" NOT NULL DEFAULT 'A',
    "name" character varying(50) NOT NULL,
    "database_action" "public"."database_action_type" NOT NULL,
    "source" "public"."table_change_type" NOT NULL,
    "table_name" character varying(255) NULL,
    "topic" character varying(255) NULL,
    "description" text NULL,
    "email_profile_id" uuid NULL,
    "email_recipients" text NULL,
    "conditional_logic" jsonb NULL,
    "custom_subject" character varying(255) NULL,
    "function_name" character varying(50) NULL,
    "trigger_name" character varying(50) NULL,
    "listener_name" character varying(50) NULL,
    "effective_date" date NULL,
    "expiration_date" date NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_table_change_alerts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_table_change_alerts_email_profile" FOREIGN KEY ("email_profile_id") REFERENCES "public"."email_profiles" ("id") ON UPDATE CASCADE ON DELETE SET NULL,
    CONSTRAINT "fk_table_change_alerts_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create unique index for name and organization_id with LOWER applied
CREATE UNIQUE INDEX "uni_table_change_alert_name" ON "table_change_alerts" (LOWER("name"), "organization_id");

-- Create index "idx_table_change_alerts_business_unit_id" to table: "table_change_alerts"
CREATE INDEX "idx_table_change_alerts_business_unit_id" ON "table_change_alerts" ("business_unit_id");

-- Create index "idx_table_change_alerts_organization_id" to table: "table_change_alerts"
CREATE INDEX "idx_table_change_alerts_organization_id" ON "table_change_alerts" ("organization_id");

-- Create "users" table
CREATE TABLE
  "public"."users" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "status" "public"."status_type" NOT NULL DEFAULT 'A',
    "name" character varying(255) NOT NULL,
    "username" character varying(30) NOT NULL,
    "password" character varying(100) NOT NULL,
    "email" character varying(255) NOT NULL,
    "date_joined" date NOT NULL,
    "timezone" "public"."timezone_type" NOT NULL DEFAULT 'America/Los_Angeles',
    "profile_pic_url" character varying(255) NULL,
    "thumbnail_url" character varying(255) NULL,
    "phone_number" character varying(20) NULL,
    "is_admin" boolean NOT NULL DEFAULT false,
    "is_super_admin" boolean NOT NULL DEFAULT false,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_users_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_users_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
  );

-- Create unique indexes for username, email, and phone_number with LOWER applied
CREATE UNIQUE INDEX "uni_user_organization_username" ON "users" ("organization_id", LOWER("username"));

CREATE UNIQUE INDEX "uni_user_organization_email" ON "users" ("organization_id", LOWER("email"));

CREATE UNIQUE INDEX "uni_user_organization_phone_number" ON "users" ("organization_id", LOWER("phone_number"));

-- Create index "idx_user_organization" to table: "users"
CREATE INDEX "idx_user_organization" ON "users" ("organization_id");

-- Create index "idx_user_business_unit" to table: "users"
CREATE INDEX "idx_user_business_unit" ON "users" ("business_unit_id");

-- Create index "idx_user_username" to table: "users"
CREATE INDEX "idx_user_username" ON "users" (LOWER("username"));

-- Create index "idx_user_email" to table: "users"
CREATE INDEX "idx_user_email" ON "users" (LOWER("email"));

-- Create "user_favorites" table
CREATE TABLE
  "public"."user_favorites" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NULL,
    "updated_at" timestamptz NULL,
    "organization_id" uuid NOT NULL,
    "business_unit_id" uuid NOT NULL,
    "user_id" uuid NOT NULL,
    "page_link" character varying(255) NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_user_favorites_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "public"."business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_user_favorites_organization" FOREIGN KEY ("organization_id") REFERENCES "public"."organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT "fk_user_favorites_user" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE CASCADE ON DELETE SET NULL
  );

-- Create unique index for page_link and user_id. This will prevent a user from adding the same page_link to their favorites more than once.
CREATE UNIQUE INDEX "uqx_user_favorites_user" ON "user_favorites" ("page_link", "user_id");

-- Create index "idx_user_favorites_organization" to table: "user_favorites"
CREATE INDEX "idx_user_favorites_organization" ON "user_favorites" ("organization_id");

-- Create index "idx_user_favorites_business_unit" to table: "user_favorites"
CREATE INDEX "idx_user_favorites_business_unit" ON "user_favorites" ("business_unit_id");

-- Create index "idx_user_favorites_user" to table: "user_favorites"
CREATE INDEX "idx_user_favorites_user" ON "user_favorites" ("user_id");