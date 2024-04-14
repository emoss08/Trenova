-- Modify "organizations" table
ALTER TABLE "organizations" ADD COLUMN "organization_accounting_control" uuid NULL;
-- Create "accounting_controls" table
CREATE TABLE "accounting_controls" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "rec_threshold" bigint NOT NULL DEFAULT 50, "rec_threshold_action" character varying NOT NULL DEFAULT 'Halt', "auto_create_journal_entries" boolean NOT NULL DEFAULT false, "restrict_manual_journal_entries" boolean NOT NULL DEFAULT false, "require_journal_entry_approval" boolean NOT NULL DEFAULT false, "enable_rec_notifications" boolean NOT NULL DEFAULT true, "halt_on_pending_rec" boolean NOT NULL DEFAULT false, "critical_processes" text NOT NULL, "default_rev_account_id" uuid NULL, "default_exp_account_id" uuid NULL, "organization_id" uuid NOT NULL, "business_unit_id" uuid NOT NULL, PRIMARY KEY ("id"));
-- Modify "organizations" table
ALTER TABLE "organizations" ADD CONSTRAINT "organizations_accounting_controls_accounting_control" FOREIGN KEY ("organization_accounting_control") REFERENCES "accounting_controls" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "accounting_controls" table
ALTER TABLE "accounting_controls" ADD CONSTRAINT "accounting_controls_business_units_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "accounting_controls_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
