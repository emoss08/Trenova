DROP INDEX IF EXISTS "idx_invoices_balance_due";

--bun:split
ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "balance_due_minor";

--bun:split
DROP INDEX IF EXISTS "idx_invoices_voided_by_adjustment";

--bun:split
DROP INDEX IF EXISTS "uq_invoices_active_billing_queue_item";

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "uk_invoices_billing_queue_item" UNIQUE ("billing_queue_item_id", "organization_id", "business_unit_id");

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "chk_invoices_void_disposition";

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "chk_invoices_status";

--bun:split
ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "voided_at",
    DROP COLUMN IF EXISTS "voided_by_id",
    DROP COLUMN IF EXISTS "void_reason",
    DROP COLUMN IF EXISTS "void_disposition",
    DROP COLUMN IF EXISTS "voided_by_adjustment_id";
