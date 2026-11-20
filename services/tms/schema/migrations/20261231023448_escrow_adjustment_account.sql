-- Manual escrow adjustments post against their own GL account rather than purchased
-- transportation, so a correction to an owner-operator's escrow stays visible and reconcilable
-- (49 CFR 376.12(k)) and never moves cost per mile.
ALTER TABLE "accounting_controls"
    ADD COLUMN IF NOT EXISTS "default_escrow_adjustment_account_id" varchar(100);

ALTER TABLE "accounting_controls"
    ADD CONSTRAINT "fk_accounting_controls_default_escrow_adjustment_account" FOREIGN KEY ("default_escrow_adjustment_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT;
