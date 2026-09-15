-- An invoice can now be voided. A draft is voided in place; a posted invoice is
-- voided by the full-reversal adjustment that writes its reversing entries. The
-- billing queue item behind a voided invoice is released, so the unique link
-- between item and invoice ignores voided rows.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "voided_at" bigint,
    ADD COLUMN IF NOT EXISTS "voided_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "void_reason" text,
    ADD COLUMN IF NOT EXISTS "void_disposition" varchar(20),
    ADD COLUMN IF NOT EXISTS "voided_by_adjustment_id" varchar(100);

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "chk_invoices_status" CHECK ("status" IN ('Draft', 'Posted', 'Voided'));

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "chk_invoices_void_disposition" CHECK ("void_disposition" IS NULL OR "void_disposition" IN ('Rebill', 'DoNotRebill'));

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "uk_invoices_billing_queue_item";

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_invoices_active_billing_queue_item" ON "invoices"("billing_queue_item_id", "organization_id", "business_unit_id")
WHERE
    "status" <> 'Voided';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoices_voided_by_adjustment" ON "invoices"("voided_by_adjustment_id", "organization_id", "business_unit_id")
WHERE
    "voided_by_adjustment_id" IS NOT NULL;

--bun:split
-- The open balance as a stored column, so the invoice register can filter and
-- sort on it without recomputing per row.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "balance_due_minor" bigint GENERATED ALWAYS AS (CASE WHEN "status" = 'Posted' AND "bill_type" IN ('Invoice', 'DebitMemo') THEN GREATEST("total_amount_minor" - "applied_amount_minor", 0) ELSE 0 END) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoices_balance_due" ON "invoices"("organization_id", "business_unit_id", "balance_due_minor")
WHERE
    "balance_due_minor" > 0;
