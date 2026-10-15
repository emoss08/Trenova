--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "factoring_reserve_entries";

--bun:split
DROP FUNCTION IF EXISTS prevent_factoring_reserve_entry_update();

--bun:split
UPDATE "factoring_invoice_states" SET "verdict" = 'Submitted' WHERE "verdict" = 'Funded';

--bun:split
ALTER TABLE "factoring_invoice_states"
    DROP CONSTRAINT IF EXISTS "chk_factoring_invoice_states_verdict";

--bun:split
ALTER TABLE "factoring_invoice_states"
    ADD CONSTRAINT "chk_factoring_invoice_states_verdict" CHECK ("verdict" IN ('Ready', 'NotReady', 'Submitted', 'Excluded'));

--bun:split
DROP INDEX IF EXISTS "idx_factoring_submission_items_charged_back";

--bun:split
DROP INDEX IF EXISTS "idx_factoring_submission_items_funded";

--bun:split
DROP INDEX IF EXISTS "uq_factoring_submission_items_live_invoice";

--bun:split
UPDATE "factoring_submission_items" SET "status" = 'Released', "released_at" = COALESCE("released_at", "updated_at")
WHERE "status" IN ('Funded', 'Rejected', 'Settled', 'ChargedBack');

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_submission_items_live_invoice"
    ON "factoring_submission_items"("organization_id", "business_unit_id", "invoice_id")
    WHERE "status" = 'Pending';

--bun:split
ALTER TABLE "factoring_submission_items"
    DROP CONSTRAINT IF EXISTS "chk_factoring_submission_items_actuals",
    DROP CONSTRAINT IF EXISTS "chk_factoring_submission_items_charged_back",
    DROP CONSTRAINT IF EXISTS "chk_factoring_submission_items_settled",
    DROP CONSTRAINT IF EXISTS "chk_factoring_submission_items_rejected",
    DROP CONSTRAINT IF EXISTS "chk_factoring_submission_items_funded",
    DROP CONSTRAINT IF EXISTS "chk_factoring_submission_items_status";

--bun:split
ALTER TABLE "factoring_submission_items"
    ADD CONSTRAINT "chk_factoring_submission_items_status" CHECK ("status" IN ('Pending', 'Released'));

--bun:split
ALTER TABLE "factoring_submission_items"
    DROP COLUMN IF EXISTS "chargeback_warned_at",
    DROP COLUMN IF EXISTS "chargeback_reason",
    DROP COLUMN IF EXISTS "charged_back_on",
    DROP COLUMN IF EXISTS "settled_on",
    DROP COLUMN IF EXISTS "rejection_reason",
    DROP COLUMN IF EXISTS "reserve_minor",
    DROP COLUMN IF EXISTS "fee_minor",
    DROP COLUMN IF EXISTS "advance_minor",
    DROP COLUMN IF EXISTS "funded_on";

--bun:split
DROP INDEX IF EXISTS "idx_factoring_submissions_unfunded";

--bun:split
UPDATE "factoring_submissions" SET "status" = 'Sent' WHERE "status" IN ('PartiallyFunded', 'Funded', 'Rejected');

--bun:split
ALTER TABLE "factoring_submissions"
    DROP CONSTRAINT IF EXISTS "chk_factoring_submissions_funded",
    DROP CONSTRAINT IF EXISTS "chk_factoring_submissions_status";

--bun:split
ALTER TABLE "factoring_submissions"
    ADD CONSTRAINT "chk_factoring_submissions_status" CHECK ("status" IN ('Sending', 'Sent', 'Failed', 'Cancelled'));

--bun:split
ALTER TABLE "factoring_submissions"
    DROP COLUMN IF EXISTS "funded_at";

--bun:split
ALTER TABLE "factoring_agreements"
    DROP CONSTRAINT IF EXISTS "fk_factoring_agreements_cash_account",
    DROP CONSTRAINT IF EXISTS "fk_factoring_agreements_liability_account",
    DROP CONSTRAINT IF EXISTS "fk_factoring_agreements_fee_expense_account",
    DROP CONSTRAINT IF EXISTS "fk_factoring_agreements_due_from_factor_account";

--bun:split
ALTER TABLE "factoring_agreements"
    DROP COLUMN IF EXISTS "cash_account_id",
    DROP COLUMN IF EXISTS "factoring_liability_account_id",
    DROP COLUMN IF EXISTS "fee_expense_account_id",
    DROP COLUMN IF EXISTS "due_from_factor_account_id";
