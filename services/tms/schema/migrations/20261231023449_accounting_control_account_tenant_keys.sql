-- Every default GL account on an accounting control names an account of the same organization and
-- business unit. A stored id that names no account of the control's own tenant is cleared first,
-- so the account has to be chosen again in Accounting Controls.
UPDATE "accounting_controls" ac
SET "default_cash_account_id" = NULL
WHERE ac."default_cash_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_cash_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_unapplied_cash_account_id" = NULL
WHERE ac."default_unapplied_cash_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_unapplied_cash_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_driver_pay_expense_account_id" = NULL
WHERE ac."default_driver_pay_expense_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_driver_pay_expense_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_purchased_transportation_account_id" = NULL
WHERE ac."default_purchased_transportation_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_purchased_transportation_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_settlements_payable_account_id" = NULL
WHERE ac."default_settlements_payable_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_settlements_payable_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_driver_advance_account_id" = NULL
WHERE ac."default_driver_advance_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_driver_advance_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_escrow_liability_account_id" = NULL
WHERE ac."default_escrow_liability_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_escrow_liability_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_driver_reimbursement_account_id" = NULL
WHERE ac."default_driver_reimbursement_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_driver_reimbursement_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

UPDATE "accounting_controls" ac
SET "default_escrow_interest_expense_account_id" = NULL
WHERE ac."default_escrow_interest_expense_account_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM "gl_accounts" g
        WHERE g."id" = ac."default_escrow_interest_expense_account_id"
            AND g."organization_id" = ac."organization_id"
            AND g."business_unit_id" = ac."business_unit_id");

ALTER TABLE "accounting_controls"
    ADD CONSTRAINT "fk_accounting_controls_default_cash_account" FOREIGN KEY ("default_cash_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_unapplied_cash_account" FOREIGN KEY ("default_unapplied_cash_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_driver_pay_expense_account" FOREIGN KEY ("default_driver_pay_expense_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_purchased_transportation_account" FOREIGN KEY ("default_purchased_transportation_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_settlements_payable_account" FOREIGN KEY ("default_settlements_payable_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_driver_advance_account" FOREIGN KEY ("default_driver_advance_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_escrow_liability_account" FOREIGN KEY ("default_escrow_liability_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_driver_reimbursement_account" FOREIGN KEY ("default_driver_reimbursement_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_accounting_controls_default_escrow_interest_expense_account" FOREIGN KEY ("default_escrow_interest_expense_account_id", "organization_id", "business_unit_id")
        REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT;
