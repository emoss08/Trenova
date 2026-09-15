-- An invoice may leave by email, by EDI 210, or both. The customer's billing
-- profile says which channels are open; the invoice records where its 210
-- stands separately from its email, and the message it produced points back.
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "email_invoice_enabled" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "edi_invoice_enabled" boolean NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "edi_send_status" varchar(30) NOT NULL DEFAULT 'NotSent',
    ADD COLUMN IF NOT EXISTS "last_edi_message_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "edi_sent_at" bigint,
    ADD COLUMN IF NOT EXISTS "last_edi_error" text;

--bun:split
ALTER TABLE "invoices"
    ADD CONSTRAINT "chk_invoices_edi_send_status" CHECK ("edi_send_status" IN ('NotSent', 'NotConfigured', 'Queued', 'Generated', 'Sending', 'Sent', 'Failed', 'DeadLettered'));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_invoices_edi_send_status" ON "invoices"("organization_id", "business_unit_id", "edi_send_status")
WHERE
    "edi_send_status" <> 'NotSent';

--bun:split
ALTER TABLE "edi_messages"
    ADD COLUMN IF NOT EXISTS "invoice_id" varchar(100);

--bun:split
ALTER TABLE "edi_messages"
    ADD CONSTRAINT "fk_edi_messages_invoice" FOREIGN KEY ("invoice_id", "organization_id", "business_unit_id") REFERENCES "invoices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_messages_invoice" ON "edi_messages"("invoice_id", "organization_id", "business_unit_id")
WHERE
    "invoice_id" IS NOT NULL;
