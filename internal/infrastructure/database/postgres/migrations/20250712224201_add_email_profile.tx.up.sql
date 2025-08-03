--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "email_provider_type_enum" AS ENUM(
    'SMTP',
    'SendGrid',
    'AWS_SES',
    'Mailgun',
    'Postmark',
    'Exchange',
    'Office365'
);

--bun:split
CREATE TYPE "email_auth_type_enum" AS ENUM(
    'Plain',
    'Login',
    'CRAMMD5',
    'OAuth2',
    'APIKey'
);

--bun:split
CREATE TYPE "email_encryption_type_enum" AS ENUM(
    'None',
    'SSL_TLS',
    'StartTLS'
);

--bun:split
CREATE TYPE "email_template_category_enum" AS ENUM(
    'Transactional',
    'Notification',
    'Marketing',
    'System',
    'Custom'
);

--bun:split
CREATE TYPE "email_priority_enum" AS ENUM(
    'High',
    'Medium',
    'Low'
);

--bun:split
CREATE TYPE "email_queue_status_enum" AS ENUM(
    'Pending',
    'Processing',
    'Sent',
    'Failed',
    'Scheduled',
    'Cancelled'
);

--bun:split
CREATE TYPE "email_log_status_enum" AS ENUM(
    'Delivered',
    'Opened',
    'Clicked',
    'Bounced',
    'Complained',
    'Unsubscribed',
    'Rejected'
);

--bun:split
CREATE TYPE "email_bounce_type_enum" AS ENUM(
    'Hard',
    'Soft',
    'Block'
);

--bun:split
CREATE TABLE IF NOT EXISTS "email_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "is_default" boolean DEFAULT FALSE,
    "status" status_enum DEFAULT 'Active',
    "provider_type" email_provider_type_enum NOT NULL,
    "auth_type" email_auth_type_enum NOT NULL,
    "encryption_type" email_encryption_type_enum NOT NULL,
    "host" varchar(255),
    "port" integer CHECK ("port" > 0 AND "port" <= 65535),
    "username" varchar(255),
    "encrypted_password" text,
    "encrypted_api_key" text,
    "oauth2_client_id" varchar(255),
    "oauth2_client_secret" text,
    "oauth2_tenant_id" varchar(255),
    "from_address" varchar(255) NOT NULL,
    "from_name" varchar(255),
    "reply_to" varchar(255),
    "max_connections" integer DEFAULT 5 CHECK ("max_connections" > 0 AND "max_connections" <= 100),
    "timeout_seconds" integer DEFAULT 30 CHECK ("timeout_seconds" >= 5 AND "timeout_seconds" <= 300),
    "retry_count" integer DEFAULT 3 CHECK ("retry_count" >= 0 AND "retry_count" <= 10),
    "retry_delay_seconds" integer DEFAULT 5 CHECK ("retry_delay_seconds" >= 1 AND "retry_delay_seconds" <= 60),
    "rate_limit_per_minute" integer DEFAULT 60 CHECK ("rate_limit_per_minute" > 0 AND "rate_limit_per_minute" <= 1000),
    "rate_limit_per_hour" integer DEFAULT 1000 CHECK ("rate_limit_per_hour" > 0 AND "rate_limit_per_hour" <= 100000),
    "rate_limit_per_day" integer DEFAULT 10000 CHECK ("rate_limit_per_day" > 0 AND "rate_limit_per_day" <= 1000000),
    "metadata" jsonb DEFAULT '{}',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_email_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_email_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    -- Provider-specific validations
    CONSTRAINT "chk_email_profiles_smtp_config" CHECK (CASE WHEN "provider_type" = 'SMTP' THEN
        "host" IS NOT NULL AND "port" IS NOT NULL AND ("auth_type" != 'APIKey' OR "encrypted_api_key" IS NOT NULL)
    ELSE
        TRUE
    END),
    CONSTRAINT "chk_email_profiles_api_providers" CHECK (CASE WHEN "provider_type" IN ('SendGrid', 'Mailgun', 'Postmark') THEN
        "encrypted_api_key" IS NOT NULL
    ELSE
        TRUE
    END),
    CONSTRAINT "chk_email_profiles_oauth_providers" CHECK (CASE WHEN "provider_type" IN ('Exchange', 'Office365') AND "auth_type" = 'OAuth2' THEN
        "oauth2_client_id" IS NOT NULL AND "oauth2_client_secret" IS NOT NULL AND "oauth2_tenant_id" IS NOT NULL
    ELSE
        TRUE
    END)
);

--bun:split
-- Add table comment for email_profiles
COMMENT ON TABLE "email_profiles" IS 'Stores email configuration profiles for different providers, allowing organizations to manage multiple email sending configurations';

--bun:split
-- Add column comments for email_profiles
COMMENT ON COLUMN "email_profiles"."id" IS 'Unique identifier for the email profile';

COMMENT ON COLUMN "email_profiles"."business_unit_id" IS 'Reference to the business unit this profile belongs to';

COMMENT ON COLUMN "email_profiles"."organization_id" IS 'Reference to the organization this profile belongs to';

COMMENT ON COLUMN "email_profiles"."name" IS 'Human-readable name for the email profile';

COMMENT ON COLUMN "email_profiles"."description" IS 'Optional description of the email profile purpose or usage';

COMMENT ON COLUMN "email_profiles"."is_default" IS 'Flag indicating if this is the default profile for the organization';

COMMENT ON COLUMN "email_profiles"."status" IS 'Active/Inactive status of the profile';

COMMENT ON COLUMN "email_profiles"."provider_type" IS 'Type of email provider (SMTP, SendGrid, AWS SES, etc.)';

COMMENT ON COLUMN "email_profiles"."auth_type" IS 'Authentication method used by the provider';

COMMENT ON COLUMN "email_profiles"."encryption_type" IS 'Encryption method for secure connections';

COMMENT ON COLUMN "email_profiles"."host" IS 'SMTP server hostname (for SMTP providers)';

COMMENT ON COLUMN "email_profiles"."port" IS 'SMTP server port number';

COMMENT ON COLUMN "email_profiles"."username" IS 'Username for authentication';

COMMENT ON COLUMN "email_profiles"."encrypted_password" IS 'Encrypted password for authentication';

COMMENT ON COLUMN "email_profiles"."encrypted_api_key" IS 'Encrypted API key for API-based providers';

COMMENT ON COLUMN "email_profiles"."oauth2_client_id" IS 'OAuth2 client ID for OAuth2 authentication';

COMMENT ON COLUMN "email_profiles"."oauth2_client_secret" IS 'OAuth2 client secret (encrypted)';

COMMENT ON COLUMN "email_profiles"."oauth2_tenant_id" IS 'OAuth2 tenant ID for Microsoft services';

COMMENT ON COLUMN "email_profiles"."from_address" IS 'Default sender email address';

COMMENT ON COLUMN "email_profiles"."from_name" IS 'Default sender display name';

COMMENT ON COLUMN "email_profiles"."reply_to" IS 'Default reply-to email address';

COMMENT ON COLUMN "email_profiles"."max_connections" IS 'Maximum concurrent connections allowed';

COMMENT ON COLUMN "email_profiles"."timeout_seconds" IS 'Connection timeout in seconds';

COMMENT ON COLUMN "email_profiles"."retry_count" IS 'Number of retry attempts for failed sends';

COMMENT ON COLUMN "email_profiles"."retry_delay_seconds" IS 'Delay between retry attempts in seconds';

COMMENT ON COLUMN "email_profiles"."rate_limit_per_minute" IS 'Maximum emails allowed per minute';

COMMENT ON COLUMN "email_profiles"."rate_limit_per_hour" IS 'Maximum emails allowed per hour';

COMMENT ON COLUMN "email_profiles"."rate_limit_per_day" IS 'Maximum emails allowed per day';

COMMENT ON COLUMN "email_profiles"."metadata" IS 'Additional provider-specific configuration stored as JSON';

COMMENT ON COLUMN "email_profiles"."version" IS 'Version number for optimistic locking';

COMMENT ON COLUMN "email_profiles"."created_at" IS 'Unix timestamp when the record was created';

COMMENT ON COLUMN "email_profiles"."updated_at" IS 'Unix timestamp when the record was last updated';

--bun:split
-- Create indexes for email profiles
CREATE INDEX "idx_email_profiles_org_id" ON "email_profiles"("organization_id");

CREATE INDEX "idx_email_profiles_business_unit_id" ON "email_profiles"("business_unit_id");

CREATE INDEX "idx_email_profiles_is_default" ON "email_profiles"("is_default")
WHERE
    "is_default" = TRUE;

CREATE INDEX "idx_email_profile_bu_org_id" ON "email_profiles"("business_unit_id", "organization_id", "id");

CREATE INDEX "idx_email_profiles_status" ON "email_profiles"("status")
WHERE
    "status" = 'Active';

CREATE INDEX "idx_email_profiles_provider_type" ON "email_profiles"("provider_type");

CREATE UNIQUE INDEX "uniq_email_profiles_org_name" ON "email_profiles"("organization_id", LOWER("name"));

CREATE INDEX "idx_email_profiles_created_at" ON "email_profiles"("created_at" DESC);

-- Composite index for common queries
CREATE INDEX "idx_email_profiles_org_status_default" ON "email_profiles"("organization_id", "status", "is_default")
WHERE
    "status" = 'Active';

--bun:split
-- Set statistics for frequently queried columns
ALTER TABLE "email_profiles"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "email_profiles"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

ALTER TABLE "email_profiles"
    ALTER COLUMN "status" SET STATISTICS 500;

--bun:split
CREATE TABLE IF NOT EXISTS "email_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "slug" varchar(255) NOT NULL,
    "description" text,
    "category" email_template_category_enum NOT NULL,
    "is_system" boolean DEFAULT FALSE,
    "is_active" boolean DEFAULT TRUE,
    "status" status_enum DEFAULT 'Active',
    "subject_template" text NOT NULL CHECK (LENGTH("subject_template") > 0 AND LENGTH("subject_template") <= 500),
    "html_template" text NOT NULL CHECK (LENGTH("html_template") > 0 AND LENGTH("html_template") <= 1048576),
    "text_template" text CHECK (LENGTH("text_template") <= 524288),
    "variables_schema" jsonb DEFAULT '{}',
    "metadata" jsonb DEFAULT '{}',
    "search_vector" tsvector,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_email_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_email_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    -- Slug validation
    CONSTRAINT "chk_email_templates_slug" CHECK ("slug" ~ '^[a-z0-9-]+$'),
    -- System templates cannot be inactive
    CONSTRAINT "chk_email_templates_system_active" CHECK (CASE WHEN "is_system" = TRUE THEN
        "is_active" = TRUE AND "status" = 'Active'
    ELSE
        TRUE
    END)
);

--bun:split
COMMENT ON TABLE "email_templates" IS 'Stores reusable email templates with variable substitution support for consistent email formatting';

COMMENT ON COLUMN "email_templates"."id" IS 'Unique identifier for the email template';

COMMENT ON COLUMN "email_templates"."business_unit_id" IS 'Reference to the business unit this template belongs to';

COMMENT ON COLUMN "email_templates"."organization_id" IS 'Reference to the organization this template belongs to';

COMMENT ON COLUMN "email_templates"."name" IS 'Human-readable name for the template';

COMMENT ON COLUMN "email_templates"."slug" IS 'URL-friendly unique identifier for the template';

COMMENT ON COLUMN "email_templates"."description" IS 'Description of the template purpose and usage';

COMMENT ON COLUMN "email_templates"."category" IS 'Template category (transactional, notification, marketing, etc.)';

COMMENT ON COLUMN "email_templates"."is_system" IS 'Flag indicating if this is a system template (read-only)';

COMMENT ON COLUMN "email_templates"."is_active" IS 'Flag indicating if the template is available for use';

COMMENT ON COLUMN "email_templates"."status" IS 'Active/Inactive status of the template';

COMMENT ON COLUMN "email_templates"."subject_template" IS 'Email subject line template with variable substitution support';

COMMENT ON COLUMN "email_templates"."html_template" IS 'HTML version of the email template';

COMMENT ON COLUMN "email_templates"."text_template" IS 'Plain text version of the email template';

COMMENT ON COLUMN "email_templates"."variables_schema" IS 'JSON schema defining available variables and their types';

COMMENT ON COLUMN "email_templates"."metadata" IS 'Additional template metadata stored as JSON';

COMMENT ON COLUMN "email_templates"."search_vector" IS 'Full-text search vector for template search';

COMMENT ON COLUMN "email_templates"."version" IS 'Version number for optimistic locking';

COMMENT ON COLUMN "email_templates"."created_at" IS 'Unix timestamp when the record was created';

COMMENT ON COLUMN "email_templates"."updated_at" IS 'Unix timestamp when the record was last updated';

--bun:split
-- Create indexes for email templates
CREATE INDEX "idx_email_templates_org_id" ON "email_templates"("organization_id");

CREATE INDEX "idx_email_templates_business_unit_id" ON "email_templates"("business_unit_id");

CREATE INDEX "idx_email_templates_category" ON "email_templates"("category");

CREATE INDEX "idx_email_templates_is_system" ON "email_templates"("is_system")
WHERE
    "is_system" = TRUE;

CREATE INDEX "idx_email_templates_is_active" ON "email_templates"("is_active")
WHERE
    "is_active" = TRUE;

CREATE INDEX "idx_email_templates_status" ON "email_templates"("status")
WHERE
    "status" = 'Active';

CREATE INDEX "idx_email_templates_search" ON "email_templates" USING gin("search_vector");

CREATE UNIQUE INDEX "uniq_email_templates_org_slug" ON "email_templates"("organization_id", LOWER("slug"));

-- Composite indexes for common queries
CREATE INDEX "idx_email_templates_active_templates" ON "email_templates"("organization_id", "category", "status")
WHERE
    "is_active" = TRUE AND "status" = 'Active';

-- Trigram index for name search
CREATE INDEX "idx_email_templates_name_trgm" ON "email_templates" USING gin("name" gin_trgm_ops);

--bun:split
-- Set statistics for frequently queried columns
ALTER TABLE "email_templates"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "email_templates"
    ALTER COLUMN "category" SET STATISTICS 500;

ALTER TABLE "email_templates"
    ALTER COLUMN "status" SET STATISTICS 500;

--bun:split
CREATE OR REPLACE FUNCTION update_email_template_search_vector()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.slug, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') || setweight(to_tsvector('english', COALESCE(NEW.category::text, '')), 'C');
    -- Auto-update timestamp
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE TRIGGER "email_templates_search_update"
    BEFORE INSERT OR UPDATE ON "email_templates"
    FOR EACH ROW
    EXECUTE FUNCTION update_email_template_search_vector();

--bun:split
CREATE TABLE IF NOT EXISTS "email_queues"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "profile_id" varchar(100) NOT NULL,
    "template_id" varchar(100),
    "to_addresses" text[] NOT NULL CHECK (array_length("to_addresses", 1) > 0),
    "cc_addresses" text[],
    "bcc_addresses" text[],
    "subject" text NOT NULL CHECK (LENGTH("subject") > 0 AND LENGTH("subject") <= 500),
    "html_body" text CHECK (LENGTH("html_body") <= 10485760), -- 10MB limit
    "text_body" text CHECK (LENGTH("text_body") <= 5242880), -- 5MB limit
    "attachments" jsonb DEFAULT '[]',
    "priority" email_priority_enum DEFAULT 'Medium',
    "status" email_queue_status_enum DEFAULT 'Pending',
    "scheduled_at" bigint,
    "sent_at" bigint,
    "error_message" text,
    "retry_count" integer DEFAULT 0 CHECK ("retry_count" >= 0 AND "retry_count" <= 10),
    "template_variables" jsonb DEFAULT '{}',
    "metadata" jsonb DEFAULT '{}',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_email_queues" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_email_queue_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_queue_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_queue_profile" FOREIGN KEY ("profile_id", "organization_id", "business_unit_id") REFERENCES "email_profiles"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_queue_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "email_templates"("id", "organization_id", "business_unit_id") ON DELETE SET NULL,
    -- Either HTML or text body must be present
    CONSTRAINT "chk_email_queue_body" CHECK ("html_body" IS NOT NULL OR "text_body" IS NOT NULL),
    -- Scheduled emails must have a scheduled_at timestamp
    CONSTRAINT "chk_email_queue_scheduled" CHECK (CASE WHEN "status" = 'Scheduled' THEN
        "scheduled_at" IS NOT NULL
    ELSE
        TRUE
    END),
    -- Sent emails must have sent_at timestamp
    CONSTRAINT "chk_email_queue_sent" CHECK (CASE WHEN "status" = 'Sent' THEN
        "sent_at" IS NOT NULL
    ELSE
        TRUE
    END),
    -- Failed emails should have error message
    CONSTRAINT "chk_email_queue_failed" CHECK (CASE WHEN "status" = 'Failed' THEN
        "error_message" IS NOT NULL
    ELSE
        TRUE
    END)
);

--bun:split
COMMENT ON TABLE "email_queues" IS 'Queue for outgoing emails with support for scheduling, retry logic, and priority-based processing';

COMMENT ON COLUMN "email_queues"."id" IS 'Unique identifier for the queued email';

COMMENT ON COLUMN "email_queues"."organization_id" IS 'Reference to the organization sending the email';

COMMENT ON COLUMN "email_queues"."business_unit_id" IS 'Reference to the business unit sending the email';

COMMENT ON COLUMN "email_queues"."profile_id" IS 'Reference to the email profile to use for sending';

COMMENT ON COLUMN "email_queues"."template_id" IS 'Optional reference to email template used';

COMMENT ON COLUMN "email_queues"."to_addresses" IS 'Array of recipient email addresses';

COMMENT ON COLUMN "email_queues"."cc_addresses" IS 'Array of CC recipient email addresses';

COMMENT ON COLUMN "email_queues"."bcc_addresses" IS 'Array of BCC recipient email addresses';

COMMENT ON COLUMN "email_queues"."subject" IS 'Email subject line';

COMMENT ON COLUMN "email_queues"."html_body" IS 'HTML version of the email body';

COMMENT ON COLUMN "email_queues"."text_body" IS 'Plain text version of the email body';

COMMENT ON COLUMN "email_queues"."attachments" IS 'JSON array of attachment metadata';

COMMENT ON COLUMN "email_queues"."priority" IS 'Priority level for queue processing (high, medium, low)';

COMMENT ON COLUMN "email_queues"."status" IS 'Current status of the email in the queue';

COMMENT ON COLUMN "email_queues"."scheduled_at" IS 'Unix timestamp for scheduled sending (null for immediate)';

COMMENT ON COLUMN "email_queues"."sent_at" IS 'Unix timestamp when the email was successfully sent';

COMMENT ON COLUMN "email_queues"."error_message" IS 'Error message if sending failed';

COMMENT ON COLUMN "email_queues"."retry_count" IS 'Number of retry attempts made';

COMMENT ON COLUMN "email_queues"."template_variables" IS 'Variables used for template substitution';

COMMENT ON COLUMN "email_queues"."metadata" IS 'Additional email metadata stored as JSON';

COMMENT ON COLUMN "email_queues"."created_at" IS 'Unix timestamp when the record was created';

COMMENT ON COLUMN "email_queues"."updated_at" IS 'Unix timestamp when the record was last updated';

--bun:split
-- Create indexes for email queue
CREATE INDEX "idx_email_queue_org_id" ON "email_queues"("organization_id");

CREATE INDEX "idx_email_queue_business_unit_id" ON "email_queues"("business_unit_id");

CREATE INDEX "idx_email_queue_profile_id" ON "email_queues"("profile_id");

CREATE INDEX "idx_email_queue_template_id" ON "email_queues"("template_id")
WHERE
    "template_id" IS NOT NULL;

CREATE INDEX "idx_email_queue_status" ON "email_queues"("status");

CREATE INDEX "idx_email_queue_priority" ON "email_queues"("priority");

CREATE INDEX "idx_email_queue_scheduled_at" ON "email_queues"("scheduled_at")
WHERE
    "scheduled_at" IS NOT NULL;

CREATE INDEX "idx_email_queue_created_at" ON "email_queues"("created_at" DESC);

-- Composite index for pending emails
CREATE INDEX "idx_email_queue_pending" ON "email_queues"("status", "priority" DESC, "created_at")
WHERE
    "status" IN ('Pending', 'Scheduled');

-- Index for retry logic
CREATE INDEX "idx_email_queue_retry" ON "email_queues"("status", "retry_count", "updated_at")
WHERE
    "status" = 'Failed' AND "retry_count" < 10;

-- BRIN index for time-based queries
CREATE INDEX "idx_email_queue_dates_brin" ON "email_queues" USING BRIN("created_at", "scheduled_at", "sent_at") WITH (pages_per_range = 128);

--bun:split
-- Set statistics for frequently queried columns
ALTER TABLE "email_queues"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "email_queues"
    ALTER COLUMN "status" SET STATISTICS 1000;

ALTER TABLE "email_queues"
    ALTER COLUMN "priority" SET STATISTICS 500;

--bun:split
CREATE TABLE IF NOT EXISTS "email_logs"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "queue_id" varchar(100) NOT NULL,
    "message_id" varchar(255),
    "status" email_log_status_enum NOT NULL,
    "provider_response" text,
    "opened_at" bigint,
    "clicked_at" bigint,
    "bounced_at" bigint,
    "complained_at" bigint,
    "unsubscribed_at" bigint,
    "bounce_type" email_bounce_type_enum,
    "bounce_reason" text,
    "webhook_events" jsonb DEFAULT '[]',
    "user_agent" text,
    "ip_address" inet,
    "clicked_urls" text[],
    "metadata" jsonb DEFAULT '{}',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_email_logs" PRIMARY KEY ("id", "queue_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_email_logs_queue" FOREIGN KEY ("queue_id", "organization_id", "business_unit_id") REFERENCES "email_queues"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_logs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_email_logs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    -- Bounce type requires bounce timestamp
    CONSTRAINT "chk_email_logs_bounce" CHECK (CASE WHEN "bounce_type" IS NOT NULL THEN
        "bounced_at" IS NOT NULL
    ELSE
        TRUE
    END),
    -- Status validations
    CONSTRAINT "chk_email_logs_status_timestamps" CHECK (CASE WHEN "status" = 'Opened' THEN
        "opened_at" IS NOT NULL
    WHEN "status" = 'Clicked' THEN
        "clicked_at" IS NOT NULL
    WHEN "status" = 'Bounced' THEN
        "bounced_at" IS NOT NULL
    WHEN "status" = 'Complained' THEN
        "complained_at" IS NOT NULL
    WHEN "status" = 'Unsubscribed' THEN
        "unsubscribed_at" IS NOT NULL
    ELSE
        TRUE
    END)
);

--bun:split
-- Add table comment for email_logs
COMMENT ON TABLE "email_logs" IS 'Logs email delivery status and engagement metrics including opens, clicks, bounces, and complaints';

COMMENT ON COLUMN "email_logs"."id" IS 'Unique identifier for the log entry';

COMMENT ON COLUMN "email_logs"."organization_id" IS 'Reference to the organization';

COMMENT ON COLUMN "email_logs"."business_unit_id" IS 'Reference to the business unit';

COMMENT ON COLUMN "email_logs"."queue_id" IS 'Reference to the email queue entry';

COMMENT ON COLUMN "email_logs"."message_id" IS 'Provider-specific message ID for tracking';

COMMENT ON COLUMN "email_logs"."status" IS 'Current delivery status of the email';

COMMENT ON COLUMN "email_logs"."provider_response" IS 'Raw response from the email provider';

COMMENT ON COLUMN "email_logs"."opened_at" IS 'Unix timestamp when the email was first opened';

COMMENT ON COLUMN "email_logs"."clicked_at" IS 'Unix timestamp when a link was first clicked';

COMMENT ON COLUMN "email_logs"."bounced_at" IS 'Unix timestamp when the email bounced';

COMMENT ON COLUMN "email_logs"."complained_at" IS 'Unix timestamp when a complaint was received';

COMMENT ON COLUMN "email_logs"."unsubscribed_at" IS 'Unix timestamp when recipient unsubscribed';

COMMENT ON COLUMN "email_logs"."bounce_type" IS 'Type of bounce (hard, soft, block)';

COMMENT ON COLUMN "email_logs"."bounce_reason" IS 'Detailed reason for the bounce';

COMMENT ON COLUMN "email_logs"."webhook_events" IS 'JSON array of webhook events from provider';

COMMENT ON COLUMN "email_logs"."user_agent" IS 'User agent string from email open/click events';

COMMENT ON COLUMN "email_logs"."ip_address" IS 'IP address from email open/click events';

COMMENT ON COLUMN "email_logs"."clicked_urls" IS 'Array of URLs that were clicked';

COMMENT ON COLUMN "email_logs"."metadata" IS 'Additional tracking metadata stored as JSON';

COMMENT ON COLUMN "email_logs"."created_at" IS 'Unix timestamp when the record was created';

--bun:split
-- Create indexes for email logs
CREATE INDEX "idx_email_logs_queue_id" ON "email_logs"("queue_id");

CREATE INDEX "idx_email_logs_message_id" ON "email_logs"("message_id")
WHERE
    "message_id" IS NOT NULL;

CREATE INDEX "idx_email_logs_status" ON "email_logs"("status");

CREATE INDEX "idx_email_logs_created_at" ON "email_logs"("created_at" DESC);

-- Event-specific indexes
CREATE INDEX "idx_email_logs_opened" ON "email_logs"("opened_at")
WHERE
    "opened_at" IS NOT NULL;

CREATE INDEX "idx_email_logs_clicked" ON "email_logs"("clicked_at")
WHERE
    "clicked_at" IS NOT NULL;

CREATE INDEX "idx_email_logs_bounced" ON "email_logs"("bounced_at", "bounce_type")
WHERE
    "bounced_at" IS NOT NULL;

-- Composite index for analytics
CREATE INDEX "idx_email_logs_analytics" ON "email_logs"("organization_id", "status", "created_at" DESC);

-- BRIN index for time-based queries
CREATE INDEX "idx_email_logs_events_brin" ON "email_logs" USING BRIN("created_at", "opened_at", "clicked_at", "bounced_at") WITH (pages_per_range = 128);

--bun:split
-- Set statistics for frequently queried columns
ALTER TABLE "email_logs"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

ALTER TABLE "email_logs"
    ALTER COLUMN "queue_id" SET STATISTICS 1000;

ALTER TABLE "email_logs"
    ALTER COLUMN "status" SET STATISTICS 1000;

--bun:split
-- Function to ensure only one default email profile per organization
CREATE OR REPLACE FUNCTION "ensure_single_default_email_profile"()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF NEW.is_default = TRUE THEN
        UPDATE
            email_profiles
        SET
            is_default = FALSE,
            updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
        WHERE
            organization_id = NEW.organization_id
            AND id != NEW.id
            AND is_default = TRUE;
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE TRIGGER "email_profiles_default_check"
    AFTER INSERT OR UPDATE OF is_default ON email_profiles
    FOR EACH ROW
    WHEN(NEW.is_default = TRUE)
    EXECUTE FUNCTION ensure_single_default_email_profile();

--bun:split
-- Function to update email queue status with timestamp tracking
CREATE OR REPLACE FUNCTION update_email_queue_status()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- Update timestamp based on status change
    IF NEW.status != OLD.status THEN
        NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
        IF NEW.status = 'Sent' AND NEW.sent_at IS NULL THEN
            NEW.sent_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
        END IF;
    END IF;
    -- Increment retry count on status change from Failed to Pending
    IF OLD.status = 'Failed' AND NEW.status = 'Pending' THEN
        NEW.retry_count := COALESCE(NEW.retry_count, 0) + 1;
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE TRIGGER "email_queue_status_update"
    BEFORE UPDATE ON "email_queues"
    FOR EACH ROW
    EXECUTE FUNCTION update_email_queue_status();

--bun:split
-- Function to prevent modification of system templates
CREATE OR REPLACE FUNCTION protect_system_templates()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF OLD.is_system = TRUE AND(NEW.name != OLD.name OR NEW.slug != OLD.slug OR NEW.subject_template != OLD.subject_template OR NEW.html_template != OLD.html_template OR NEW.text_template != OLD.text_template OR NEW.is_system != OLD.is_system OR NEW.is_active != OLD.is_active OR NEW.status != OLD.status) THEN
        RAISE EXCEPTION 'System templates cannot be modified';
    END IF;
    IF TG_OP = 'DELETE' AND OLD.is_system = TRUE THEN
        RAISE EXCEPTION 'System templates cannot be deleted';
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
CREATE TRIGGER "protect_system_templates_update"
    BEFORE UPDATE ON "email_templates"
    FOR EACH ROW
    EXECUTE FUNCTION protect_system_templates();

CREATE TRIGGER "protect_system_templates_delete"
    BEFORE DELETE ON "email_templates"
    FOR EACH ROW
    EXECUTE FUNCTION protect_system_templates();

--bun:split
-- Create views for common queries
CREATE OR REPLACE VIEW email_queue_summary AS
SELECT
    eq.organization_id,
    eq.business_unit_id,
    eq.status,
    eq.priority,
    DATE(to_timestamp(eq.created_at)) AS queue_date,
    COUNT(*) AS email_count,
    COUNT(
        CASE WHEN eq.status = 'Failed' THEN
            1
        END) AS failed_count,
    COUNT(
        CASE WHEN eq.status = 'Sent' THEN
            1
        END) AS sent_count,
    AVG(
        CASE WHEN eq.sent_at IS NOT NULL THEN
            eq.sent_at - eq.created_at
        END) AS avg_send_time_seconds
FROM
    email_queues eq
GROUP BY
    eq.organization_id,
    eq.business_unit_id,
    eq.status,
    eq.priority,
    queue_date;

COMMENT ON VIEW email_queue_summary IS 'Aggregated view of email queue statistics by organization, status, and date';

--bun:split
CREATE OR REPLACE VIEW email_engagement_metrics AS
SELECT
    el.organization_id,
    el.business_unit_id,
    DATE(to_timestamp(el.created_at)) AS log_date,
    COUNT(DISTINCT el.queue_id) AS total_emails,
    COUNT(DISTINCT CASE WHEN el.status = 'Delivered' THEN
            el.queue_id
        END) AS delivered_count,
    COUNT(DISTINCT CASE WHEN el.opened_at IS NOT NULL THEN
            el.queue_id
        END) AS opened_count,
    COUNT(DISTINCT CASE WHEN el.clicked_at IS NOT NULL THEN
            el.queue_id
        END) AS clicked_count,
    COUNT(DISTINCT CASE WHEN el.bounced_at IS NOT NULL THEN
            el.queue_id
        END) AS bounced_count,
    COUNT(DISTINCT CASE WHEN el.bounce_type = 'Hard' THEN
            el.queue_id
        END) AS hard_bounce_count,
    COUNT(DISTINCT CASE WHEN el.complained_at IS NOT NULL THEN
            el.queue_id
        END) AS complaint_count,
    COUNT(DISTINCT CASE WHEN el.unsubscribed_at IS NOT NULL THEN
            el.queue_id
        END) AS unsubscribe_count
FROM
    email_logs el
GROUP BY
    el.organization_id,
    el.business_unit_id,
    log_date;

COMMENT ON VIEW email_engagement_metrics IS 'Email engagement metrics aggregated by organization and date';

