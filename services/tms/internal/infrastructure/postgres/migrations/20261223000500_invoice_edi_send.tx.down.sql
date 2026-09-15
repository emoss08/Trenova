DROP INDEX IF EXISTS "idx_edi_messages_invoice";

--bun:split
ALTER TABLE "edi_messages"
    DROP CONSTRAINT IF EXISTS "fk_edi_messages_invoice";

--bun:split
ALTER TABLE "edi_messages"
    DROP COLUMN IF EXISTS "invoice_id";

--bun:split
DROP INDEX IF EXISTS "idx_invoices_edi_send_status";

--bun:split
ALTER TABLE "invoices"
    DROP CONSTRAINT IF EXISTS "chk_invoices_edi_send_status";

--bun:split
ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "edi_send_status",
    DROP COLUMN IF EXISTS "last_edi_message_id",
    DROP COLUMN IF EXISTS "edi_sent_at",
    DROP COLUMN IF EXISTS "last_edi_error";

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "email_invoice_enabled",
    DROP COLUMN IF EXISTS "edi_invoice_enabled";
