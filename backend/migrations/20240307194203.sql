-- Drop index "idx_accounting_controls_default_expense_account_id" from table: "accounting_controls"
DROP INDEX "public"."idx_accounting_controls_default_expense_account_id";
-- Drop index "idx_accounting_controls_default_revenue_account_id" from table: "accounting_controls"
DROP INDEX "public"."idx_accounting_controls_default_revenue_account_id";
-- Drop index "idx_accounting_controls_organization_id" from table: "accounting_controls"
DROP INDEX "public"."idx_accounting_controls_organization_id";
-- Modify "accounting_controls" table
ALTER TABLE "public"."accounting_controls" DROP CONSTRAINT "accounting_control_reconciliation_threshold_check", DROP CONSTRAINT "accounting_controls_organization_id_key";
-- Drop index "idx_business_unit_parent_id" from table: "business_units"
DROP INDEX "public"."idx_business_unit_parent_id";
-- Drop index "uni_business_units_entity_key" from table: "business_units"
DROP INDEX "public"."uni_business_units_entity_key";
-- Drop index "uni_business_units_name" from table: "business_units"
DROP INDEX "public"."uni_business_units_name";
-- Create index "uni_business_units_entity_key" to table: "business_units"
CREATE UNIQUE INDEX "uni_business_units_entity_key" ON "public"."business_units" ("entity_key");
-- Create index "uni_business_units_name" to table: "business_units"
CREATE UNIQUE INDEX "uni_business_units_name" ON "public"."business_units" ("name");
-- Drop index "idx_division_code_ap_account_id" from table: "division_codes"
DROP INDEX "public"."idx_division_code_ap_account_id";
-- Drop index "idx_division_code_business_unit" from table: "division_codes"
DROP INDEX "public"."idx_division_code_business_unit";
-- Drop index "idx_division_code_cash_account_id" from table: "division_codes"
DROP INDEX "public"."idx_division_code_cash_account_id";
-- Drop index "idx_division_code_code" from table: "division_codes"
DROP INDEX "public"."idx_division_code_code";
-- Drop index "idx_division_code_expense_account_id" from table: "division_codes"
DROP INDEX "public"."idx_division_code_expense_account_id";
-- Drop index "idx_division_code_organization" from table: "division_codes"
DROP INDEX "public"."idx_division_code_organization";
-- Drop index "uni_division_code_organization_code" from table: "division_codes"
DROP INDEX "public"."uni_division_code_organization_code";
-- Drop index "idx_general_ledger_account_tags_general_ledger_account_id" from table: "general_ledger_account_tags"
DROP INDEX "public"."idx_general_ledger_account_tags_general_ledger_account_id";
-- Drop index "idx_general_ledger_account_tags_tag_id" from table: "general_ledger_account_tags"
DROP INDEX "public"."idx_general_ledger_account_tags_tag_id";
-- Drop index "idx_general_ledger_account_account_number" from table: "general_ledger_accounts"
DROP INDEX "public"."idx_general_ledger_account_account_number";
-- Drop index "idx_general_ledger_account_business_unit" from table: "general_ledger_accounts"
DROP INDEX "public"."idx_general_ledger_account_business_unit";
-- Drop index "idx_general_ledger_account_organization" from table: "general_ledger_accounts"
DROP INDEX "public"."idx_general_ledger_account_organization";
-- Drop index "uni_general_ledger_account_organization" from table: "general_ledger_accounts"
DROP INDEX "public"."uni_general_ledger_account_organization";
-- Drop index "idx_job_title_business_unit" from table: "job_titles"
DROP INDEX "public"."idx_job_title_business_unit";
-- Drop index "idx_job_title_name" from table: "job_titles"
DROP INDEX "public"."idx_job_title_name";
-- Drop index "idx_job_title_organization" from table: "job_titles"
DROP INDEX "public"."idx_job_title_organization";
-- Drop index "uni_job_title_organization" from table: "job_titles"
DROP INDEX "public"."uni_job_title_organization";
-- Drop index "idx_organizations_business_unit" from table: "organizations"
DROP INDEX "public"."idx_organizations_business_unit";
-- Drop index "uni_organizations_name" from table: "organizations"
DROP INDEX "public"."uni_organizations_name";
-- Create index "uni_organizations_name" to table: "organizations"
CREATE UNIQUE INDEX "uni_organizations_name" ON "public"."organizations" ("name");
-- Drop index "idx_revenue_code_business_unit" from table: "revenue_codes"
DROP INDEX "public"."idx_revenue_code_business_unit";
-- Drop index "idx_revenue_code_code" from table: "revenue_codes"
DROP INDEX "public"."idx_revenue_code_code";
-- Drop index "idx_revenue_code_expense_account_id" from table: "revenue_codes"
DROP INDEX "public"."idx_revenue_code_expense_account_id";
-- Drop index "idx_revenue_code_organization" from table: "revenue_codes"
DROP INDEX "public"."idx_revenue_code_organization";
-- Drop index "idx_revenue_code_revenue_account_id" from table: "revenue_codes"
DROP INDEX "public"."idx_revenue_code_revenue_account_id";
-- Drop index "uni_revenue_code_organization" from table: "revenue_codes"
DROP INDEX "public"."uni_revenue_code_organization";
-- Drop index "idx_table_change_alerts_business_unit_id" from table: "table_change_alerts"
DROP INDEX "public"."idx_table_change_alerts_business_unit_id";
-- Drop index "idx_table_change_alerts_organization_id" from table: "table_change_alerts"
DROP INDEX "public"."idx_table_change_alerts_organization_id";
-- Drop index "uni_table_change_alert_name" from table: "table_change_alerts"
DROP INDEX "public"."uni_table_change_alert_name";
-- Drop index "tag_organization_uqx" from table: "tags"
DROP INDEX "public"."tag_organization_uqx";
-- Drop index "idx_user_favorites_business_unit" from table: "user_favorites"
DROP INDEX "public"."idx_user_favorites_business_unit";
-- Drop index "idx_user_favorites_organization" from table: "user_favorites"
DROP INDEX "public"."idx_user_favorites_organization";
-- Drop index "idx_user_favorites_user" from table: "user_favorites"
DROP INDEX "public"."idx_user_favorites_user";
-- Drop index "uqx_user_favorites_user" from table: "user_favorites"
DROP INDEX "public"."uqx_user_favorites_user";
-- Drop index "idx_user_business_unit" from table: "users"
DROP INDEX "public"."idx_user_business_unit";
-- Drop index "idx_user_email" from table: "users"
DROP INDEX "public"."idx_user_email";
-- Drop index "idx_user_organization" from table: "users"
DROP INDEX "public"."idx_user_organization";
-- Drop index "idx_user_username" from table: "users"
DROP INDEX "public"."idx_user_username";
-- Drop index "uni_user_organization_email" from table: "users"
DROP INDEX "public"."uni_user_organization_email";
-- Drop index "uni_user_organization_phone_number" from table: "users"
DROP INDEX "public"."uni_user_organization_phone_number";
-- Drop index "uni_user_organization_username" from table: "users"
DROP INDEX "public"."uni_user_organization_username";
