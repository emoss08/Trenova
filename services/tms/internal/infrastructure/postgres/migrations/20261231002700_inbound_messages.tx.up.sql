-- An address we listen on.
--
-- The token is in the URL the provider posts to and is the only thing that
-- identifies the tenant, so it is stored hashed: a leaked row is something
-- somebody can read, not an address they can post to.
CREATE TABLE IF NOT EXISTS "inbound_mailboxes"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "address" varchar(255) NOT NULL,
    "provider" varchar(20) NOT NULL,
    "token_hash" varchar(128) NOT NULL,
    "purpose" varchar(255),
    "review_policy" varchar(30) NOT NULL DEFAULT 'AlwaysReview',
    "min_confidence" numeric(4, 3) NOT NULL DEFAULT 0,
    "status" varchar(20) NOT NULL DEFAULT 'Active',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_inbound_mailboxes" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_mailboxes_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_mailboxes_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_mailboxes_provider" CHECK ("provider" IN ('Postmark', 'Resend')),
    CONSTRAINT "ck_inbound_mailboxes_review_policy" CHECK ("review_policy" IN ('AlwaysReview', 'ReviewBelowConfidence', 'AutoHandle')),
    CONSTRAINT "ck_inbound_mailboxes_status" CHECK ("status" IN ('Active', 'Inactive'))
);

--bun:split
-- The token is how a delivery finds its tenant, so it has to resolve in one
-- indexed read and it has to be unique across every organization.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_mailboxes_token" ON "inbound_mailboxes"("token_hash");

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_mailboxes_address" ON "inbound_mailboxes"("organization_id", "business_unit_id", lower("address"));

--bun:split
-- One email that arrived, kept whether or not anything could be made of it.
-- A tender nobody matched is still what somebody has to look at, and a message
-- dropped for not classifying is one the sender believes was received.
CREATE TABLE IF NOT EXISTS "inbound_messages"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "mailbox_id" varchar(100) NOT NULL,
    "provider_message_id" varchar(255) NOT NULL,
    "message_id" varchar(500),
    "in_reply_to" varchar(500),
    "references" jsonb,
    "from_address" varchar(255) NOT NULL,
    "from_name" varchar(255),
    "to_addresses" jsonb,
    "cc_addresses" jsonb,
    "subject" varchar(500),
    "text_body" text,
    "html_key" varchar(512),
    "received_at" bigint NOT NULL,
    "spam_score" numeric(6, 3),
    "classification" varchar(30),
    "confidence" numeric(4, 3),
    "status" varchar(20) NOT NULL DEFAULT 'Received',
    "matched_customer_id" varchar(100),
    "matched_carrier_id" varchar(100),
    "matched_shipment_id" varchar(100),
    "match_reason" varchar(500),
    "run_id" varchar(100),
    "reviewed_by" varchar(100),
    "reviewed_at" bigint,
    "review_note" text,
    "failure_code" varchar(100),
    "failure_text" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_inbound_messages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_messages_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_messages_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_messages_mailbox" FOREIGN KEY ("mailbox_id", "business_unit_id", "organization_id") REFERENCES "inbound_mailboxes"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_messages_classification" CHECK ("classification" IS NULL OR "classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other')),
    CONSTRAINT "ck_inbound_messages_status" CHECK ("status" IN ('Received', 'Processing', 'Classified', 'InReview', 'Actioned', 'Ignored', 'Quarantined'))
);

--bun:split
-- A provider redelivering a webhook is ordinary, and without this it is a
-- second shipment off the same tender.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_messages_provider" ON "inbound_messages"("mailbox_id", "provider_message_id");

--bun:split
-- The inbox reads one lane at a time, newest first.
CREATE INDEX IF NOT EXISTS "idx_inbound_messages_lane" ON "inbound_messages"("organization_id", "business_unit_id", "status", "received_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_inbound_messages_shipment" ON "inbound_messages"("organization_id", "business_unit_id", "matched_shipment_id") WHERE "matched_shipment_id" IS NOT NULL;

--bun:split
-- A reply has to carry the thread's headers or it starts a new conversation in
-- the sender's client, which is how a customer ends up with two threads about
-- one load.
CREATE INDEX IF NOT EXISTS "idx_inbound_messages_thread" ON "inbound_messages"("organization_id", "business_unit_id", "message_id") WHERE "message_id" IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS "inbound_message_attachments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "message_id" varchar(100) NOT NULL,
    "file_name" varchar(255) NOT NULL,
    "content_type" varchar(100),
    "byte_size" bigint,
    "kind" varchar(30) NOT NULL DEFAULT 'Unknown',
    "upload_session_id" varchar(100),
    "document_id" varchar(100),
    "draft_id" varchar(100),
    "failure_text" text,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_inbound_message_attachments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_message_attachments_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_attachments_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_attachments_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_message_attachments_kind" CHECK ("kind" IN ('Unknown', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'BillOfLading', 'Other'))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_inbound_message_attachments_message" ON "inbound_message_attachments"("message_id", "business_unit_id", "organization_id");

COMMENT ON TABLE "inbound_mailboxes" IS 'Addresses the system listens on, with how much each is trusted to act without a person';

COMMENT ON COLUMN "inbound_mailboxes"."token_hash" IS 'The webhook token as stored; the token itself is shown once and never again';

COMMENT ON COLUMN "inbound_messages"."match_reason" IS 'Why the message was matched to these records — a match nobody can check is one nobody will trust';
