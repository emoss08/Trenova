-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260530120000_invoice_delivery.tx.up.sql

ALTER TABLE "documents" ADD COLUMN "crypto_mode" TEXT NOT NULL DEFAULT 'envelope_v1';

--bun:split

ALTER TABLE "documents" ADD COLUMN "crypto_version" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "document_upload_sessions" ADD COLUMN "checksum_sha256" TEXT;

--bun:split

ALTER TABLE "document_upload_sessions" ADD COLUMN "crypto_mode" TEXT NOT NULL DEFAULT 'envelope_v1';

--bun:split

ALTER TABLE "document_upload_sessions" ADD COLUMN "crypto_version" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "pdf_document_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "send_status" TEXT NOT NULL DEFAULT 'NotSent';

--bun:split

ALTER TABLE "invoices" ADD COLUMN "sent_at" INTEGER;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "sent_by_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "last_send_error" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "last_send_warning" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "memo" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "remittance_instructions" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "email_subject_snapshot" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "email_body_snapshot" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "email_to_snapshot" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "email_cc_snapshot" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "email_bcc_snapshot" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS invoice_attachments(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "selected" INTEGER NOT NULL DEFAULT 1,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_attachments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_attachments_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_attachments_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT uk_invoice_attachments_document UNIQUE (invoice_id, document_id, organization_id, business_unit_id)
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_document_share_tokens(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "expires_at" INTEGER NOT NULL,
    "downloaded_at" INTEGER,
    "revoked_at" INTEGER,
    "created_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_document_share_tokens PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_document_share_tokens_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_document_share_tokens_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT uk_invoice_document_share_tokens_hash UNIQUE (token_hash)
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_email_attempts(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "email_message_id" TEXT,
    "attempt_number" INTEGER NOT NULL,
    "part_number" INTEGER NOT NULL,
    "total_parts" INTEGER NOT NULL,
    "status" TEXT NOT NULL,
    "provider" TEXT,
    "provider_message_id" TEXT,
    "to_recipients" TEXT NOT NULL,
    "cc_recipients" TEXT,
    "bcc_recipients" TEXT,
    "subject" TEXT NOT NULL,
    "body" TEXT,
    "estimated_size" INTEGER NOT NULL DEFAULT 0,
    "warnings" TEXT,
    "error" TEXT,
    "sent_at" INTEGER,
    "created_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_email_attempts PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_email_attempts_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_email_attempts_message FOREIGN KEY (email_message_id, organization_id, business_unit_id)
        REFERENCES email_messages(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_email_attempt_attachments(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "attempt_id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "content_type" TEXT NOT NULL,
    "size_bytes" INTEGER NOT NULL,
    "encoded_bytes" INTEGER NOT NULL,
    "method" TEXT NOT NULL,
    "share_token_id" TEXT,
    "reason" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_email_attempt_attachments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_email_attempt_attachments_attempt FOREIGN KEY (attempt_id, organization_id, business_unit_id)
        REFERENCES invoice_email_attempts(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_email_attempt_attachments_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_invoice_email_attempt_attachments_share_token FOREIGN KEY (share_token_id, organization_id, business_unit_id)
        REFERENCES invoice_document_share_tokens(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoices_send_status ON invoices (organization_id, business_unit_id, send_status);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_attachments_invoice ON invoice_attachments (invoice_id, organization_id, business_unit_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_email_attempts_invoice ON invoice_email_attempts (invoice_id, organization_id, business_unit_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_document_share_tokens_hash ON invoice_document_share_tokens (token_hash);
