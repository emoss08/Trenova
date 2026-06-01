ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS crypto_mode varchar(32) NOT NULL DEFAULT 'envelope_v1',
    ADD COLUMN IF NOT EXISTS crypto_version smallint NOT NULL DEFAULT 1;

--bun:split
ALTER TABLE document_upload_sessions
    ADD COLUMN IF NOT EXISTS checksum_sha256 varchar(64),
    ADD COLUMN IF NOT EXISTS crypto_mode varchar(32) NOT NULL DEFAULT 'envelope_v1',
    ADD COLUMN IF NOT EXISTS crypto_version smallint NOT NULL DEFAULT 1;

--bun:split
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS pdf_document_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS send_status VARCHAR(50) NOT NULL DEFAULT 'NotSent',
    ADD COLUMN IF NOT EXISTS sent_at BIGINT,
    ADD COLUMN IF NOT EXISTS sent_by_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS last_send_error TEXT,
    ADD COLUMN IF NOT EXISTS last_send_warning TEXT,
    ADD COLUMN IF NOT EXISTS memo TEXT,
    ADD COLUMN IF NOT EXISTS remittance_instructions TEXT,
    ADD COLUMN IF NOT EXISTS email_subject_snapshot VARCHAR(998),
    ADD COLUMN IF NOT EXISTS email_body_snapshot TEXT,
    ADD COLUMN IF NOT EXISTS email_to_snapshot TEXT[],
    ADD COLUMN IF NOT EXISTS email_cc_snapshot TEXT[],
    ADD COLUMN IF NOT EXISTS email_bcc_snapshot TEXT[];

--bun:split
ALTER TABLE invoices
    ADD CONSTRAINT fk_invoices_pdf_document
        FOREIGN KEY (pdf_document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id)
        ON DELETE SET NULL;

--bun:split
CREATE TABLE IF NOT EXISTS invoice_attachments(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    invoice_id VARCHAR(100) NOT NULL,
    document_id VARCHAR(100) NOT NULL,
    selected BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_attachments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_attachments_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_attachments_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT uk_invoice_attachments_document UNIQUE (invoice_id, document_id, organization_id, business_unit_id)
);

--bun:split
CREATE TABLE IF NOT EXISTS invoice_document_share_tokens(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    invoice_id VARCHAR(100) NOT NULL,
    document_id VARCHAR(100) NOT NULL,
    token_hash VARCHAR(128) NOT NULL,
    expires_at BIGINT NOT NULL,
    downloaded_at BIGINT,
    revoked_at BIGINT,
    created_by_id VARCHAR(100),
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_document_share_tokens PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_document_share_tokens_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_document_share_tokens_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT uk_invoice_document_share_tokens_hash UNIQUE (token_hash)
);

--bun:split
CREATE TABLE IF NOT EXISTS invoice_email_attempts(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    invoice_id VARCHAR(100) NOT NULL,
    email_message_id VARCHAR(100),
    attempt_number INTEGER NOT NULL,
    part_number INTEGER NOT NULL,
    total_parts INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    provider VARCHAR(50),
    provider_message_id VARCHAR(160),
    to_recipients TEXT[] NOT NULL,
    cc_recipients TEXT[],
    bcc_recipients TEXT[],
    subject VARCHAR(998) NOT NULL,
    body TEXT,
    estimated_size BIGINT NOT NULL DEFAULT 0,
    warnings TEXT[],
    error TEXT,
    sent_at BIGINT,
    created_by_id VARCHAR(100),
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_email_attempts PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_email_attempts_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_email_attempts_message FOREIGN KEY (email_message_id, organization_id, business_unit_id)
        REFERENCES email_messages(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split
CREATE TABLE IF NOT EXISTS invoice_email_attempt_attachments(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    attempt_id VARCHAR(100) NOT NULL,
    document_id VARCHAR(100) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(120) NOT NULL,
    size_bytes BIGINT NOT NULL,
    encoded_bytes BIGINT NOT NULL,
    method VARCHAR(50) NOT NULL,
    share_token_id VARCHAR(100),
    reason TEXT,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_email_attempt_attachments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_email_attempt_attachments_attempt FOREIGN KEY (attempt_id, organization_id, business_unit_id)
        REFERENCES invoice_email_attempts(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_email_attempt_attachments_document FOREIGN KEY (document_id, organization_id, business_unit_id)
        REFERENCES documents(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_invoice_email_attempt_attachments_share_token FOREIGN KEY (share_token_id, organization_id, business_unit_id)
        REFERENCES invoice_document_share_tokens(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoices_send_status ON invoices(organization_id, business_unit_id, send_status);
CREATE INDEX IF NOT EXISTS idx_invoice_attachments_invoice ON invoice_attachments(invoice_id, organization_id, business_unit_id);
CREATE INDEX IF NOT EXISTS idx_invoice_email_attempts_invoice ON invoice_email_attempts(invoice_id, organization_id, business_unit_id);
CREATE INDEX IF NOT EXISTS idx_invoice_document_share_tokens_hash ON invoice_document_share_tokens(token_hash);
