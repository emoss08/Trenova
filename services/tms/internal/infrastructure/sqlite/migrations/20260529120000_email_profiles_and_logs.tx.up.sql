-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260529120000_email_profiles_and_logs.tx.up.sql

CREATE TABLE IF NOT EXISTS email_profile_assignments (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "purpose" TEXT NOT NULL,
    "profile_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_email_profile_assignments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_profile_assignments_profile FOREIGN KEY (profile_id, organization_id, business_unit_id) REFERENCES email_profiles(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT uq_email_profile_assignments_purpose UNIQUE (organization_id, business_unit_id, purpose)
);

--bun:split

CREATE TABLE IF NOT EXISTS email_messages (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "profile_id" TEXT NOT NULL,
    "purpose" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "idempotency_key" TEXT NOT NULL,
    "provider_message_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Queued',
    "attempts" INTEGER NOT NULL DEFAULT 0,
    "from_email" TEXT NOT NULL,
    "from_name" TEXT NOT NULL,
    "reply_to_email" TEXT,
    "to_recipients" TEXT NOT NULL,
    "cc_recipients" TEXT,
    "bcc_recipients" TEXT,
    "subject" TEXT NOT NULL,
    "body_text_size" INTEGER NOT NULL DEFAULT 0,
    "body_html_size" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "sent_at" INTEGER,
    "delivered_at" INTEGER,
    "failed_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_email_messages PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_messages_profile FOREIGN KEY (profile_id, organization_id, business_unit_id) REFERENCES email_profiles(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT uq_email_messages_idempotency UNIQUE (organization_id, business_unit_id, idempotency_key)
);

--bun:split

CREATE TABLE IF NOT EXISTS email_message_attachments (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "message_id" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "content_type" TEXT NOT NULL,
    "object_key" TEXT NOT NULL,
    "size_bytes" INTEGER NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_email_message_attachments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_message_attachments_message FOREIGN KEY (message_id, organization_id, business_unit_id) REFERENCES email_messages(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS email_events (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "message_id" TEXT,
    "provider" TEXT NOT NULL,
    "provider_event_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "recipient" TEXT,
    "occurred_at" INTEGER NOT NULL,
    "raw" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_email_events PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_events_message FOREIGN KEY (message_id, organization_id, business_unit_id) REFERENCES email_messages(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT uq_email_events_provider UNIQUE (organization_id, business_unit_id, provider, provider_event_id)
);

--bun:split

CREATE TABLE IF NOT EXISTS email_suppressions (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "email_address" TEXT NOT NULL,
    "reason" TEXT NOT NULL,
    "provider" TEXT,
    "source_event_id" TEXT,
    "notes" TEXT,
    "created_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_email_suppressions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_suppressions_business_unit FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE,
    CONSTRAINT fk_email_suppressions_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_email_suppressions_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_email_suppressions_address UNIQUE (organization_id, business_unit_id, email_address)
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_email_messages_status ON email_messages (organization_id, business_unit_id, status, created_at);

--bun:split

CREATE INDEX IF NOT EXISTS idx_email_messages_provider_id ON email_messages (organization_id, business_unit_id, provider, provider_message_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_email_events_message ON email_events (organization_id, business_unit_id, message_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_email_suppressions_address ON email_suppressions (organization_id, business_unit_id, email_address);
