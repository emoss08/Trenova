DROP TABLE IF EXISTS "billingqueue_approval_run_items";

--bun:split
DROP TABLE IF EXISTS "billingqueue_approval_runs";

--bun:split
DROP TABLE IF EXISTS "billing_queue_events";

--bun:split
DROP TABLE IF EXISTS "billing_queue_issues";

--bun:split
ALTER TABLE "document_types"
    DROP COLUMN IF EXISTS "requires_signature";

--bun:split
ALTER TABLE "documents"
    DROP CONSTRAINT IF EXISTS "ck_documents_signature_status",
    DROP COLUMN IF EXISTS "signature_status",
    DROP COLUMN IF EXISTS "signed_at";

--bun:split
DROP INDEX IF EXISTS "idx_billing_queue_items_shipment_payer";

--bun:split
DROP INDEX IF EXISTS "idx_billing_queue_items_tenant_created";

--bun:split
ALTER TABLE "billing_queue_items"
    DROP CONSTRAINT IF EXISTS "ck_billing_queue_items_hold_reason_code",
    DROP COLUMN IF EXISTS "hold_reason_code",
    DROP COLUMN IF EXISTS "held_at",
    DROP COLUMN IF EXISTS "held_by_id",
    DROP COLUMN IF EXISTS "status_before_hold";
