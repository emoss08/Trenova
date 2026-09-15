DROP TABLE IF EXISTS "credit_memo_applications";

--bun:split
DROP INDEX IF EXISTS "idx_invoices_reference_invoice";

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "chk_invoices_memo_kind";

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "fk_invoices_reference_invoice";

--bun:split
ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "reference_invoice_id",
    DROP COLUMN IF EXISTS "memo_reason",
    DROP COLUMN IF EXISTS "memo_kind";
