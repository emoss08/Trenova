ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'Resend';

--bun:split
ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'AmazonSES';

--bun:split
ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'SendGrid';

--bun:split
ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'Mailgun';

--bun:split
ALTER TYPE integration_type ADD VALUE IF NOT EXISTS 'Postmark';

--bun:split
ALTER TYPE integration_category ADD VALUE IF NOT EXISTS 'Email';

--bun:split
ALTER TABLE email_profiles
    ALTER COLUMN auth_type SET DEFAULT 'APIKey',
    ALTER COLUMN encryption_type SET DEFAULT 'None';

--bun:split
ALTER TABLE email_profiles DROP CONSTRAINT IF EXISTS chk_email_profiles_api_providers;

--bun:split
ALTER TABLE email_profiles DROP CONSTRAINT IF EXISTS chk_email_profiles_oauth_providers;

--bun:split
CREATE TYPE email_purpose_enum AS ENUM (
    'General',
    'Billing',
    'Reporting',
    'Operations',
    'Authentication',
    'Notifications'
);

--bun:split
CREATE TYPE email_message_status_enum AS ENUM (
    'Queued',
    'Sending',
    'Sent',
    'Delivered',
    'Failed',
    'Bounced',
    'Complained',
    'Opened',
    'Clicked',
    'Suppressed'
);

--bun:split
CREATE TYPE email_event_type_enum AS ENUM (
    'Sent',
    'Delivered',
    'Opened',
    'Clicked',
    'Bounced',
    'Complained',
    'Failed'
);

--bun:split
CREATE TYPE email_suppression_reason_enum AS ENUM (
    'HardBounce',
    'Complaint',
    'SoftBounceLimit',
    'Manual'
);

--bun:split
CREATE TABLE IF NOT EXISTS email_profile_assignments (
    id varchar(100) NOT NULL,
    business_unit_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    purpose email_purpose_enum NOT NULL,
    profile_id varchar(100) NOT NULL,
    created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT pk_email_profile_assignments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_profile_assignments_profile FOREIGN KEY (profile_id, organization_id, business_unit_id) REFERENCES email_profiles(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT uq_email_profile_assignments_purpose UNIQUE (organization_id, business_unit_id, purpose)
);

--bun:split
CREATE TABLE IF NOT EXISTS email_messages (
    id varchar(100) NOT NULL,
    business_unit_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    profile_id varchar(100) NOT NULL,
    purpose email_purpose_enum NOT NULL,
    provider email_provider_type_enum NOT NULL,
    idempotency_key varchar(160) NOT NULL,
    provider_message_id varchar(160),
    status email_message_status_enum NOT NULL DEFAULT 'Queued',
    attempts integer NOT NULL DEFAULT 0,
    from_email varchar(320) NOT NULL,
    from_name varchar(100) NOT NULL,
    reply_to_email varchar(320),
    to_recipients text[] NOT NULL,
    cc_recipients text[],
    bcc_recipients text[],
    subject varchar(998) NOT NULL,
    body_text_size bigint NOT NULL DEFAULT 0,
    body_html_size bigint NOT NULL DEFAULT 0,
    last_error text,
    sent_at bigint,
    delivered_at bigint,
    failed_at bigint,
    created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    updated_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(subject, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(from_email, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(provider_message_id, '')), 'B')
    ) STORED,
    CONSTRAINT pk_email_messages PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_messages_profile FOREIGN KEY (profile_id, organization_id, business_unit_id) REFERENCES email_profiles(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT uq_email_messages_idempotency UNIQUE (organization_id, business_unit_id, idempotency_key)
);

--bun:split
CREATE TABLE IF NOT EXISTS email_message_attachments (
    id varchar(100) NOT NULL,
    business_unit_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    message_id varchar(100) NOT NULL,
    file_name varchar(255) NOT NULL,
    content_type varchar(120) NOT NULL,
    object_key text NOT NULL,
    size_bytes bigint NOT NULL,
    created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT pk_email_message_attachments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_message_attachments_message FOREIGN KEY (message_id, organization_id, business_unit_id) REFERENCES email_messages(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE TABLE IF NOT EXISTS email_events (
    id varchar(100) NOT NULL,
    business_unit_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    message_id varchar(100),
    provider email_provider_type_enum NOT NULL,
    provider_event_id varchar(200) NOT NULL,
    type email_event_type_enum NOT NULL,
    recipient varchar(320),
    occurred_at bigint NOT NULL,
    raw jsonb DEFAULT '{}'::jsonb,
    created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT pk_email_events PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_events_message FOREIGN KEY (message_id, organization_id, business_unit_id) REFERENCES email_messages(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT uq_email_events_provider UNIQUE (organization_id, business_unit_id, provider, provider_event_id)
);

--bun:split
CREATE TABLE IF NOT EXISTS email_suppressions (
    id varchar(100) NOT NULL,
    business_unit_id varchar(100) NOT NULL,
    organization_id varchar(100) NOT NULL,
    email_address varchar(320) NOT NULL,
    reason email_suppression_reason_enum NOT NULL,
    provider email_provider_type_enum,
    source_event_id varchar(200),
    notes text,
    created_by_id varchar(100),
    created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT pk_email_suppressions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_email_suppressions_business_unit FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE,
    CONSTRAINT fk_email_suppressions_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_email_suppressions_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_email_suppressions_address UNIQUE (organization_id, business_unit_id, email_address)
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_email_messages_search ON email_messages USING gin(search_vector);
CREATE INDEX IF NOT EXISTS idx_email_messages_status ON email_messages(organization_id, business_unit_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_email_messages_provider_id ON email_messages(organization_id, business_unit_id, provider, provider_message_id);
CREATE INDEX IF NOT EXISTS idx_email_events_message ON email_events(organization_id, business_unit_id, message_id);
CREATE INDEX IF NOT EXISTS idx_email_suppressions_address ON email_suppressions(organization_id, business_unit_id, email_address);
