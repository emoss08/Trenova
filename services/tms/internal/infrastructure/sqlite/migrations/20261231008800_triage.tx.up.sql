-- SQLite twin of 20261231008800_triage.tx.up.sql.

ALTER TABLE "inbound_messages" ADD COLUMN "message_key" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "thread_key" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "sender_domain" TEXT NOT NULL DEFAULT '';

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "owner_id" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "due_at" INTEGER;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "parked" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "snoozed_until" INTEGER;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "snooze_event" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "snooze_watch_id" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "snooze_document_type_id" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "snoozed_at" INTEGER;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "snoozed_by" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "awaiting_since" INTEGER;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "awaiting_until" INTEGER;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "nova_read" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "handled_by_rule_id" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "handled_action" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "classification_corrected" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "facts" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "conflicts" TEXT;

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "action_state" TEXT NOT NULL DEFAULT '{}';

--bun:split

ALTER TABLE "inbound_messages" ADD COLUMN "load_draft" TEXT;

--bun:split

UPDATE "inbound_messages"
SET "message_key" = NULLIF(lower(trim(trim("message_id"), '<>')), ''),
    "thread_key" = COALESCE(NULLIF(lower(trim(trim("message_id"), '<>')), ''), "id"),
    "sender_domain" = lower(substr("from_address", instr("from_address", '@') + 1)),
    "nova_read" = CASE WHEN "classification" IS NOT NULL AND "confidence" > 0 THEN 1 ELSE 0 END;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_waiting_due" ON "inbound_messages"("organization_id", "business_unit_id", "status", "due_at", "id") WHERE "parked" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_parked" ON "inbound_messages"("organization_id", "business_unit_id", "parked", "received_at") WHERE "parked" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_snoozed_until" ON "inbound_messages"("snoozed_until") WHERE "parked" = 'Snoozed' AND "snoozed_until" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_awaiting_until" ON "inbound_messages"("awaiting_until") WHERE "parked" = 'AwaitingReply';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_snooze_event" ON "inbound_messages"("organization_id", "business_unit_id", "snooze_event", "snooze_watch_id") WHERE "snooze_event" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_owner" ON "inbound_messages"("organization_id", "business_unit_id", "owner_id", "status", "due_at") WHERE "owner_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_thread_key" ON "inbound_messages"("organization_id", "business_unit_id", "thread_key", "received_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_message_key" ON "inbound_messages"("organization_id", "business_unit_id", "message_key") WHERE "message_key" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_mailbox_lane" ON "inbound_messages"("organization_id", "business_unit_id", "mailbox_id", "status", "received_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_kind_due" ON "inbound_messages"("organization_id", "business_unit_id", "classification", "status", "due_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_sender" ON "inbound_messages"("organization_id", "business_unit_id", "sender_domain", "classification", "received_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_rule" ON "inbound_messages"("organization_id", "business_unit_id", "handled_by_rule_id", "received_at") WHERE "handled_by_rule_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_settled_at" ON "inbound_messages"("organization_id", "business_unit_id", "reviewed_at") WHERE "status" IN ('Actioned', 'Ignored');

--bun:split

ALTER TABLE "inbound_message_attachments" ADD COLUMN "filed_shipment_id" TEXT;

--bun:split

ALTER TABLE "inbound_message_attachments" ADD COLUMN "filed_as" TEXT;

--bun:split

ALTER TABLE "inbound_message_attachments" ADD COLUMN "filed_at" INTEGER;

--bun:split

ALTER TABLE "inbound_message_attachments" ADD COLUMN "filed_by" TEXT;

--bun:split

ALTER TABLE "inbound_message_attachments" ADD COLUMN "page_count" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "inbound_message_attachments" ADD COLUMN "pages" TEXT;

--bun:split

ALTER TABLE "inbound_message_attachments" ADD COLUMN "split_documents" TEXT;

--bun:split

ALTER TABLE "inbound_mailboxes" ADD COLUMN "last_received_at" INTEGER;

--bun:split

ALTER TABLE "inbound_mailboxes" ADD COLUMN "test_token" TEXT;

--bun:split

ALTER TABLE "inbound_mailboxes" ADD COLUMN "test_sent_at" INTEGER;

--bun:split

ALTER TABLE "inbound_mailboxes" ADD COLUMN "test_received_at" INTEGER;

--bun:split

ALTER TABLE "inbound_mailboxes" ADD COLUMN "blocked_senders" TEXT;

--bun:split

UPDATE "inbound_mailboxes"
SET "last_received_at" = (
    SELECT MAX(m."received_at") FROM "inbound_messages" AS m
    WHERE m."mailbox_id" = "inbound_mailboxes"."id"
        AND m."organization_id" = "inbound_mailboxes"."organization_id"
        AND m."business_unit_id" = "inbound_mailboxes"."business_unit_id"
);

--bun:split

CREATE TABLE IF NOT EXISTS "inbound_message_activities"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "message_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "actor_kind" TEXT NOT NULL,
    "actor_id" TEXT,
    "subject" TEXT,
    "detail" TEXT,
    "until" INTEGER,
    "from_classification" TEXT,
    "to_classification" TEXT,
    "record_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_inbound_message_activities" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_message_activities_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_activities_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_activities_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_message_activities_actor_kind" CHECK ("actor_kind" IN ('Person', 'Nova', 'System'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_message_activities_message" ON "inbound_message_activities"("organization_id", "business_unit_id", "message_id", "created_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_message_activities_kind" ON "inbound_message_activities"("organization_id", "business_unit_id", "kind", "created_at");

--bun:split

CREATE TABLE IF NOT EXISTS "inbound_replies"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "message_id" TEXT NOT NULL,
    "mailbox_id" TEXT NOT NULL,
    "thread_key" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "from_address" TEXT NOT NULL,
    "to_address" TEXT NOT NULL,
    "cc_addresses" TEXT,
    "subject" TEXT NOT NULL,
    "body" TEXT NOT NULL,
    "drafted_by_nova" INTEGER NOT NULL DEFAULT 0,
    "template_id" TEXT,
    "message_key" TEXT,
    "provider_message_id" TEXT,
    "failure_text" TEXT,
    "suggested_address" TEXT,
    "idempotency_key" TEXT,
    "created_by_id" TEXT,
    "sent_by_id" TEXT,
    "sent_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_inbound_replies" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_replies_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_replies_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_replies_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_replies_status" CHECK ("status" IN ('Draft', 'Sending', 'Sent', 'Failed'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_replies_message" ON "inbound_replies"("organization_id", "business_unit_id", "message_id", "created_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_replies_message_key" ON "inbound_replies"("organization_id", "business_unit_id", "message_key") WHERE "message_key" IS NOT NULL;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_replies_idempotency" ON "inbound_replies"("organization_id", "business_unit_id", "idempotency_key") WHERE "idempotency_key" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_replies_sent" ON "inbound_replies"("organization_id", "business_unit_id", "sent_at") WHERE "status" = 'Sent';

--bun:split

CREATE TABLE IF NOT EXISTS "triage_rules"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "classification" TEXT NOT NULL,
    "sender_scope" TEXT NOT NULL,
    "sender" TEXT,
    "mailbox_id" TEXT,
    "action" TEXT NOT NULL,
    "assignee_id" TEXT,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_by_id" TEXT,
    "match_count" INTEGER NOT NULL DEFAULT 0,
    "last_matched_at" INTEGER,
    "last_message_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_triage_rules" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_triage_rules_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_rules_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_rules_mailbox" FOREIGN KEY ("mailbox_id", "business_unit_id", "organization_id") REFERENCES "inbound_mailboxes"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_triage_rules_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other')),
    CONSTRAINT "ck_triage_rules_sender_scope" CHECK ("sender_scope" IN ('Anyone', 'AnyCustomer', 'AnyCarrier', 'Domain', 'Address')),
    CONSTRAINT "ck_triage_rules_action" CHECK ("action" IN ('FileOnShipment', 'FileAndQueuePay', 'DraftReply', 'DraftLoad', 'LinkBySender', 'Ignore', 'Assign'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_triage_rules_lookup" ON "triage_rules"("organization_id", "business_unit_id", "enabled", "classification");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_triage_rules_sentence" ON "triage_rules"("organization_id", "business_unit_id", "classification", "sender_scope", COALESCE("sender", ''), COALESCE("mailbox_id", ''), "action");

--bun:split

CREATE TABLE IF NOT EXISTS "triage_reply_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "body" TEXT NOT NULL,
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_triage_reply_templates" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_triage_reply_templates_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_reply_templates_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_triage_reply_templates_name" ON "triage_reply_templates"("organization_id", "business_unit_id", lower("name"));

--bun:split

CREATE TABLE IF NOT EXISTS "triage_response_targets"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "classification" TEXT NOT NULL,
    "minutes" INTEGER NOT NULL,
    "updated_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_triage_response_targets" PRIMARY KEY ("organization_id", "business_unit_id", "classification"),
    CONSTRAINT "fk_triage_response_targets_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_response_targets_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_triage_response_targets_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other')),
    CONSTRAINT "ck_triage_response_targets_minutes" CHECK ("minutes" BETWEEN 5 AND 43200)
);

--bun:split

CREATE TABLE IF NOT EXISTS "triage_settings"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "on_call_user_id" TEXT,
    "end_of_day_minute" INTEGER NOT NULL DEFAULT 1020,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_triage_settings" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "fk_triage_settings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_settings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_triage_settings_end_of_day" CHECK ("end_of_day_minute" BETWEEN 0 AND 1439)
);

--bun:split

CREATE TABLE IF NOT EXISTS "triage_views"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "query" TEXT NOT NULL,
    "position" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_triage_views" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_triage_views_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_views_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_triage_views_user" ON "triage_views"("organization_id", "business_unit_id", "user_id", "position");

--bun:split

CREATE TABLE IF NOT EXISTS "inbound_sender_hints"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "sender_domain" TEXT NOT NULL,
    "classification" TEXT NOT NULL,
    "corrections" INTEGER NOT NULL DEFAULT 0,
    "last_corrected_at" INTEGER NOT NULL,
    CONSTRAINT "pk_inbound_sender_hints" PRIMARY KEY ("organization_id", "business_unit_id", "sender_domain", "classification"),
    CONSTRAINT "fk_inbound_sender_hints_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_sender_hints_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_sender_hints_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other'))
);
