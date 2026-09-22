-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231002700_inbound_messages.tx.up.sql

CREATE TABLE IF NOT EXISTS "inbound_mailboxes"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "address" TEXT NOT NULL,
    "provider" TEXT NOT NULL,
    "token_hash" TEXT NOT NULL,
    "purpose" TEXT,
    "review_policy" TEXT NOT NULL DEFAULT 'AlwaysReview',
    "min_confidence" REAL NOT NULL DEFAULT 0,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_inbound_mailboxes" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_mailboxes_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_mailboxes_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_mailboxes_provider" CHECK ("provider" IN ('Postmark', 'Resend')),
    CONSTRAINT "ck_inbound_mailboxes_review_policy" CHECK ("review_policy" IN ('AlwaysReview', 'ReviewBelowConfidence', 'AutoHandle')),
    CONSTRAINT "ck_inbound_mailboxes_status" CHECK ("status" IN ('Active', 'Inactive'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_mailboxes_token" ON "inbound_mailboxes" ("token_hash");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_mailboxes_address" ON "inbound_mailboxes" ("organization_id", "business_unit_id", lower("address"));

--bun:split

CREATE TABLE IF NOT EXISTS "inbound_messages"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "mailbox_id" TEXT NOT NULL,
    "provider_message_id" TEXT NOT NULL,
    "message_id" TEXT,
    "in_reply_to" TEXT,
    "references" TEXT,
    "from_address" TEXT NOT NULL,
    "from_name" TEXT,
    "to_addresses" TEXT,
    "cc_addresses" TEXT,
    "subject" TEXT,
    "text_body" TEXT,
    "html_key" TEXT,
    "received_at" INTEGER NOT NULL,
    "spam_score" REAL,
    "classification" TEXT,
    "confidence" REAL,
    "status" TEXT NOT NULL DEFAULT 'Received',
    "matched_customer_id" TEXT,
    "matched_carrier_id" TEXT,
    "matched_shipment_id" TEXT,
    "match_reason" TEXT,
    "run_id" TEXT,
    "reviewed_by" TEXT,
    "reviewed_at" INTEGER,
    "review_note" TEXT,
    "failure_code" TEXT,
    "failure_text" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_inbound_messages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_messages_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_messages_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_messages_mailbox" FOREIGN KEY ("mailbox_id", "business_unit_id", "organization_id") REFERENCES "inbound_mailboxes"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_messages_classification" CHECK ("classification" IS NULL OR "classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other')),
    CONSTRAINT "ck_inbound_messages_status" CHECK ("status" IN ('Received', 'Processing', 'Classified', 'InReview', 'Actioned', 'Ignored', 'Quarantined'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_messages_provider" ON "inbound_messages" ("mailbox_id", "provider_message_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_lane" ON "inbound_messages" ("organization_id", "business_unit_id", "status", "received_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_shipment" ON "inbound_messages" ("organization_id", "business_unit_id", "matched_shipment_id")WHERE "matched_shipment_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_thread" ON "inbound_messages" ("organization_id", "business_unit_id", "message_id")WHERE "message_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "inbound_message_attachments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "message_id" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "content_type" TEXT,
    "byte_size" INTEGER,
    "kind" TEXT NOT NULL DEFAULT 'Unknown',
    "upload_session_id" TEXT,
    "document_id" TEXT,
    "draft_id" TEXT,
    "failure_text" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_inbound_message_attachments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_message_attachments_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_attachments_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_attachments_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_message_attachments_kind" CHECK ("kind" IN ('Unknown', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'BillOfLading', 'Other'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_message_attachments_message" ON "inbound_message_attachments" ("message_id", "business_unit_id", "organization_id");
