--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- The accounts a factor's journals post to. Funding refuses until the ones its
-- treatment needs are set; cash falls back to the organization's default.
ALTER TABLE "factoring_agreements"
    ADD COLUMN IF NOT EXISTS "due_from_factor_account_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "fee_expense_account_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "factoring_liability_account_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "cash_account_id" varchar(100);

ALTER TABLE "factoring_agreements"
    ADD CONSTRAINT "fk_factoring_agreements_due_from_factor_account" FOREIGN KEY ("due_from_factor_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_factoring_agreements_fee_expense_account" FOREIGN KEY ("fee_expense_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_factoring_agreements_liability_account" FOREIGN KEY ("factoring_liability_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_factoring_agreements_cash_account" FOREIGN KEY ("cash_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT;

ALTER TABLE "factoring_submissions"
    ADD COLUMN IF NOT EXISTS "funded_at" bigint;

ALTER TABLE "factoring_submissions"
    DROP CONSTRAINT IF EXISTS "chk_factoring_submissions_status";

ALTER TABLE "factoring_submissions"
    ADD CONSTRAINT "chk_factoring_submissions_status" CHECK ("status" IN ('Sending', 'Sent', 'Failed', 'Cancelled', 'PartiallyFunded', 'Funded', 'Rejected'));

ALTER TABLE "factoring_submissions"
    ADD CONSTRAINT "chk_factoring_submissions_funded" CHECK ("status" NOT IN ('PartiallyFunded', 'Funded') OR "funded_at" IS NOT NULL);

CREATE INDEX IF NOT EXISTS "idx_factoring_submissions_unfunded"
    ON "factoring_submissions"("sent_at")
    WHERE "status" IN ('Sent', 'PartiallyFunded');

-- What the factor actually did with each invoice. The expected figures stay as
-- they were planned; these are what it paid, charged and held.
ALTER TABLE "factoring_submission_items"
    ADD COLUMN IF NOT EXISTS "funded_on" bigint,
    ADD COLUMN IF NOT EXISTS "advance_minor" bigint,
    ADD COLUMN IF NOT EXISTS "fee_minor" bigint,
    ADD COLUMN IF NOT EXISTS "reserve_minor" bigint,
    ADD COLUMN IF NOT EXISTS "rejection_reason" varchar(500),
    ADD COLUMN IF NOT EXISTS "settled_on" bigint,
    ADD COLUMN IF NOT EXISTS "charged_back_on" bigint,
    ADD COLUMN IF NOT EXISTS "chargeback_reason" varchar(500),
    ADD COLUMN IF NOT EXISTS "chargeback_warned_at" bigint;

ALTER TABLE "factoring_submission_items"
    DROP CONSTRAINT IF EXISTS "chk_factoring_submission_items_status";

ALTER TABLE "factoring_submission_items"
    ADD CONSTRAINT "chk_factoring_submission_items_status" CHECK ("status" IN ('Pending', 'Released', 'Funded', 'Rejected', 'Settled', 'ChargedBack'));

ALTER TABLE "factoring_submission_items"
    ADD CONSTRAINT "chk_factoring_submission_items_funded" CHECK ("status" NOT IN ('Funded', 'Settled', 'ChargedBack') OR ("funded_on" IS NOT NULL AND "advance_minor" IS NOT NULL AND "fee_minor" IS NOT NULL AND "reserve_minor" IS NOT NULL)),
    ADD CONSTRAINT "chk_factoring_submission_items_rejected" CHECK ("status" <> 'Rejected' OR "rejection_reason" IS NOT NULL),
    ADD CONSTRAINT "chk_factoring_submission_items_settled" CHECK ("status" <> 'Settled' OR "settled_on" IS NOT NULL),
    ADD CONSTRAINT "chk_factoring_submission_items_charged_back" CHECK ("status" <> 'ChargedBack' OR "charged_back_on" IS NOT NULL),
    ADD CONSTRAINT "chk_factoring_submission_items_actuals" CHECK (("advance_minor" IS NULL OR "advance_minor" >= 0) AND ("fee_minor" IS NULL OR "fee_minor" >= 0) AND ("reserve_minor" IS NULL OR "reserve_minor" >= 0) AND ("advance_minor" IS NULL OR "advance_minor" + "fee_minor" + "reserve_minor" = "face_minor"));

DROP INDEX IF EXISTS "uq_factoring_submission_items_live_invoice";

CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_submission_items_live_invoice"
    ON "factoring_submission_items"("organization_id", "business_unit_id", "invoice_id")
    WHERE "status" IN ('Pending', 'Funded');

CREATE INDEX IF NOT EXISTS "idx_factoring_submission_items_funded"
    ON "factoring_submission_items"("organization_id", "business_unit_id", "funded_on")
    WHERE "status" = 'Funded';

CREATE INDEX IF NOT EXISTS "idx_factoring_submission_items_charged_back"
    ON "factoring_submission_items"("organization_id", "business_unit_id", "invoice_id")
    WHERE "status" = 'ChargedBack';

-- One row per movement posted to the books for a factored invoice, beside the
-- journal entry that posted it. Summing a column gives the matching account's
-- balance from factoring journals: reserve_minor is the amount due from the
-- factor, liability_minor what is owed to it under secured borrowing.
CREATE TABLE IF NOT EXISTS "factoring_reserve_entries"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "factoring_company_id" varchar(100) NOT NULL,
    "agreement_id" varchar(100) NOT NULL,
    "submission_id" varchar(100) NOT NULL,
    "submission_item_id" varchar(100) NOT NULL,
    "invoice_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "treatment" varchar(30) NOT NULL,
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "cash_minor" bigint NOT NULL DEFAULT 0,
    "fee_minor" bigint NOT NULL DEFAULT 0,
    "reserve_minor" bigint NOT NULL DEFAULT 0,
    "liability_minor" bigint NOT NULL DEFAULT 0,
    "ar_minor" bigint NOT NULL DEFAULT 0,
    "journal_entry_id" varchar(100) NOT NULL,
    "journal_batch_id" varchar(100) NOT NULL,
    "accounting_date" bigint NOT NULL,
    "occurred_on" bigint NOT NULL,
    "reverses_entry_id" varchar(100),
    "memo" varchar(500),
    "created_by_id" varchar(100),
    "created_at" bigint NOT NULL,
    CONSTRAINT "pk_factoring_reserve_entries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_factoring_reserve_entries_factor" FOREIGN KEY ("factoring_company_id", "organization_id", "business_unit_id") REFERENCES "factoring_companies"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_reserve_entries_agreement" FOREIGN KEY ("agreement_id", "organization_id", "business_unit_id") REFERENCES "factoring_agreements"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_reserve_entries_submission" FOREIGN KEY ("submission_id", "organization_id", "business_unit_id") REFERENCES "factoring_submissions"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_reserve_entries_item" FOREIGN KEY ("submission_item_id", "organization_id", "business_unit_id") REFERENCES "factoring_submission_items"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_reserve_entries_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_reserve_entries_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_reserve_entries_journal_entry" FOREIGN KEY ("journal_entry_id", "organization_id", "business_unit_id") REFERENCES "journal_entries"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_factoring_reserve_entries_reverses" FOREIGN KEY ("reverses_entry_id", "organization_id", "business_unit_id") REFERENCES "factoring_reserve_entries"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "chk_factoring_reserve_entries_kind" CHECK ("kind" IN ('Funding', 'ReserveRelease', 'Chargeback', 'ChargebackReversal')),
    CONSTRAINT "chk_factoring_reserve_entries_treatment" CHECK ("treatment" IN ('Sale', 'SecuredBorrowing')),
    CONSTRAINT "chk_factoring_reserve_entries_reverses" CHECK (("kind" = 'ChargebackReversal') = ("reverses_entry_id" IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS "idx_factoring_reserve_entries_factor"
    ON "factoring_reserve_entries"("organization_id", "business_unit_id", "factoring_company_id", "occurred_on" DESC, "id" DESC);

CREATE INDEX IF NOT EXISTS "idx_factoring_reserve_entries_item"
    ON "factoring_reserve_entries"("organization_id", "business_unit_id", "submission_item_id");

CREATE INDEX IF NOT EXISTS "idx_factoring_reserve_entries_invoice"
    ON "factoring_reserve_entries"("organization_id", "business_unit_id", "invoice_id");

CREATE UNIQUE INDEX IF NOT EXISTS "uq_factoring_reserve_entries_reversal"
    ON "factoring_reserve_entries"("organization_id", "business_unit_id", "reverses_entry_id")
    WHERE "reverses_entry_id" IS NOT NULL;

-- A posted movement is corrected by a new one, never edited.
CREATE OR REPLACE FUNCTION prevent_factoring_reserve_entry_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    RAISE EXCEPTION 'factoring_reserve_entries is append-only; post a correcting movement instead';
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS "trg_factoring_reserve_entries_append_only" ON "factoring_reserve_entries";

CREATE TRIGGER "trg_factoring_reserve_entries_append_only"
    BEFORE UPDATE ON "factoring_reserve_entries"
    FOR EACH ROW
    EXECUTE FUNCTION prevent_factoring_reserve_entry_update();

ALTER TABLE "factoring_invoice_states"
    DROP CONSTRAINT IF EXISTS "chk_factoring_invoice_states_verdict";

ALTER TABLE "factoring_invoice_states"
    ADD CONSTRAINT "chk_factoring_invoice_states_verdict" CHECK ("verdict" IN ('Ready', 'NotReady', 'Submitted', 'Funded', 'Excluded'));

SELECT trenova_rls.reconcile();
