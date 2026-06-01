DROP TABLE IF EXISTS invoice_email_attempt_attachments;
DROP TABLE IF EXISTS invoice_email_attempts;
DROP TABLE IF EXISTS invoice_document_share_tokens;
DROP TABLE IF EXISTS invoice_attachments;

--bun:split
ALTER TABLE invoices
    DROP CONSTRAINT IF EXISTS fk_invoices_pdf_document;

--bun:split
ALTER TABLE invoices
    DROP COLUMN IF EXISTS pdf_document_id,
    DROP COLUMN IF EXISTS send_status,
    DROP COLUMN IF EXISTS sent_at,
    DROP COLUMN IF EXISTS sent_by_id,
    DROP COLUMN IF EXISTS last_send_error,
    DROP COLUMN IF EXISTS last_send_warning,
    DROP COLUMN IF EXISTS memo,
    DROP COLUMN IF EXISTS remittance_instructions,
    DROP COLUMN IF EXISTS email_subject_snapshot,
    DROP COLUMN IF EXISTS email_body_snapshot,
    DROP COLUMN IF EXISTS email_to_snapshot,
    DROP COLUMN IF EXISTS email_cc_snapshot,
    DROP COLUMN IF EXISTS email_bcc_snapshot;
