ALTER TABLE "inbound_messages"
    ADD COLUMN IF NOT EXISTS "message_key" varchar(500),
    ADD COLUMN IF NOT EXISTS "thread_key" varchar(500) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "sender_domain" varchar(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "owner_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "due_at" bigint,
    ADD COLUMN IF NOT EXISTS "parked" varchar(20),
    ADD COLUMN IF NOT EXISTS "snoozed_until" bigint,
    ADD COLUMN IF NOT EXISTS "snooze_event" varchar(30),
    ADD COLUMN IF NOT EXISTS "snooze_watch_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "snooze_document_type_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "snoozed_at" bigint,
    ADD COLUMN IF NOT EXISTS "snoozed_by" varchar(100),
    ADD COLUMN IF NOT EXISTS "awaiting_since" bigint,
    ADD COLUMN IF NOT EXISTS "awaiting_until" bigint,
    ADD COLUMN IF NOT EXISTS "nova_read" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "handled_by_rule_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "handled_action" varchar(30),
    ADD COLUMN IF NOT EXISTS "classification_corrected" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "facts" jsonb,
    ADD COLUMN IF NOT EXISTS "conflicts" jsonb,
    ADD COLUMN IF NOT EXISTS "action_state" jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "load_draft" jsonb;

ALTER TABLE "inbound_messages"
    ADD CONSTRAINT "ck_inbound_messages_parked" CHECK ("parked" IS NULL OR "parked" IN ('Snoozed', 'AwaitingReply')),
    ADD CONSTRAINT "ck_inbound_messages_snooze_event" CHECK ("snooze_event" IS NULL OR "snooze_event" IN ('ShipmentDelivered', 'DocumentAttached', 'InvoicePaid')),
    ADD CONSTRAINT "ck_inbound_messages_handled_action" CHECK ("handled_action" IS NULL OR "handled_action" IN ('FileOnShipment', 'FileAndQueuePay', 'DraftReply', 'DraftLoad', 'LinkBySender', 'Ignore', 'Assign')),
    ADD CONSTRAINT "ck_inbound_messages_parked_needs_review" CHECK ("parked" IS NULL OR "status" IN ('InReview', 'Quarantined'));

UPDATE "inbound_messages"
SET "message_key" = NULLIF(lower(btrim(btrim("message_id"), '<>')), ''),
    "thread_key" = COALESCE(NULLIF(lower(btrim(btrim("message_id"), '<>')), ''), "id"),
    "sender_domain" = lower(split_part("from_address", '@', 2)),
    "nova_read" = ("classification" IS NOT NULL AND "confidence" IS NOT NULL AND "confidence" > 0),
    "due_at" = CASE WHEN "status" IN ('InReview', 'Quarantined') THEN
        "received_at" + 60 * CASE "classification"
            WHEN 'Tender' THEN 120
            WHEN 'StatusRequest' THEN 60
            WHEN 'RateConfirmation' THEN 120
            WHEN 'ProofOfDelivery' THEN 480
            WHEN 'Invoice' THEN 1440
            WHEN 'DetentionDispute' THEN 1440
            ELSE 2880
        END
    END;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_waiting_due" ON "inbound_messages"("organization_id", "business_unit_id", "status", "due_at", "id") WHERE "parked" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_parked" ON "inbound_messages"("organization_id", "business_unit_id", "parked", "received_at" DESC) WHERE "parked" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_snoozed_until" ON "inbound_messages"("snoozed_until") WHERE "parked" = 'Snoozed' AND "snoozed_until" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_awaiting_until" ON "inbound_messages"("awaiting_until") WHERE "parked" = 'AwaitingReply';

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_snooze_event" ON "inbound_messages"("organization_id", "business_unit_id", "snooze_event", "snooze_watch_id") WHERE "snooze_event" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_owner" ON "inbound_messages"("organization_id", "business_unit_id", "owner_id", "status", "due_at") WHERE "owner_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_thread_key" ON "inbound_messages"("organization_id", "business_unit_id", "thread_key", "received_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_message_key" ON "inbound_messages"("organization_id", "business_unit_id", "message_key") WHERE "message_key" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_mailbox_lane" ON "inbound_messages"("organization_id", "business_unit_id", "mailbox_id", "status", "received_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_kind_due" ON "inbound_messages"("organization_id", "business_unit_id", "classification", "status", "due_at");

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_sender" ON "inbound_messages"("organization_id", "business_unit_id", "sender_domain", "classification", "received_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_rule" ON "inbound_messages"("organization_id", "business_unit_id", "handled_by_rule_id", "received_at" DESC) WHERE "handled_by_rule_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_settled_at" ON "inbound_messages"("organization_id", "business_unit_id", "reviewed_at" DESC) WHERE "status" IN ('Actioned', 'Ignored');

ALTER TABLE "inbound_message_attachments"
    ADD COLUMN IF NOT EXISTS "filed_shipment_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "filed_as" varchar(100),
    ADD COLUMN IF NOT EXISTS "filed_at" bigint,
    ADD COLUMN IF NOT EXISTS "filed_by" varchar(100),
    ADD COLUMN IF NOT EXISTS "page_count" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "pages" jsonb,
    ADD COLUMN IF NOT EXISTS "split_documents" jsonb;

ALTER TABLE "inbound_mailboxes"
    ADD COLUMN IF NOT EXISTS "last_received_at" bigint,
    ADD COLUMN IF NOT EXISTS "test_token" varchar(64),
    ADD COLUMN IF NOT EXISTS "test_sent_at" bigint,
    ADD COLUMN IF NOT EXISTS "test_received_at" bigint,
    ADD COLUMN IF NOT EXISTS "blocked_senders" jsonb;

UPDATE "inbound_mailboxes" AS mb
SET "last_received_at" = latest."received_at"
FROM (
    SELECT "mailbox_id", "organization_id", "business_unit_id", MAX("received_at") AS "received_at"
    FROM "inbound_messages"
    GROUP BY "mailbox_id", "organization_id", "business_unit_id"
) AS latest
WHERE latest."mailbox_id" = mb."id"
    AND latest."organization_id" = mb."organization_id"
    AND latest."business_unit_id" = mb."business_unit_id";

CREATE TABLE IF NOT EXISTS "inbound_message_activities"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "message_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "actor_kind" varchar(20) NOT NULL,
    "actor_id" varchar(100),
    "subject" varchar(500),
    "detail" varchar(500),
    "until" bigint,
    "from_classification" varchar(30),
    "to_classification" varchar(30),
    "record_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_inbound_message_activities" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_message_activities_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_activities_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_message_activities_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_message_activities_actor_kind" CHECK ("actor_kind" IN ('Person', 'Nova', 'System'))
);

CREATE INDEX IF NOT EXISTS "idx_inbound_message_activities_message" ON "inbound_message_activities"("organization_id", "business_unit_id", "message_id", "created_at");

CREATE INDEX IF NOT EXISTS "idx_inbound_message_activities_kind" ON "inbound_message_activities"("organization_id", "business_unit_id", "kind", "created_at" DESC);

CREATE TABLE IF NOT EXISTS "inbound_replies"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "message_id" varchar(100) NOT NULL,
    "mailbox_id" varchar(100) NOT NULL,
    "thread_key" varchar(500) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Draft',
    "from_address" varchar(255) NOT NULL,
    "to_address" varchar(255) NOT NULL,
    "cc_addresses" jsonb,
    "subject" varchar(500) NOT NULL,
    "body" text NOT NULL,
    "drafted_by_nova" boolean NOT NULL DEFAULT FALSE,
    "template_id" varchar(100),
    "message_key" varchar(500),
    "provider_message_id" varchar(255),
    "failure_text" text,
    "suggested_address" varchar(255),
    "idempotency_key" varchar(255),
    "created_by_id" varchar(100),
    "sent_by_id" varchar(100),
    "sent_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_inbound_replies" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_inbound_replies_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_replies_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_replies_message" FOREIGN KEY ("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_replies_status" CHECK ("status" IN ('Draft', 'Sending', 'Sent', 'Failed'))
);

CREATE INDEX IF NOT EXISTS "idx_inbound_replies_message" ON "inbound_replies"("organization_id", "business_unit_id", "message_id", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_inbound_replies_message_key" ON "inbound_replies"("organization_id", "business_unit_id", "message_key") WHERE "message_key" IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "uq_inbound_replies_idempotency" ON "inbound_replies"("organization_id", "business_unit_id", "idempotency_key") WHERE "idempotency_key" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_inbound_replies_sent" ON "inbound_replies"("organization_id", "business_unit_id", "sent_at") WHERE "status" = 'Sent';

CREATE TABLE IF NOT EXISTS "triage_rules"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "classification" varchar(30) NOT NULL,
    "sender_scope" varchar(20) NOT NULL,
    "sender" varchar(255),
    "mailbox_id" varchar(100),
    "action" varchar(30) NOT NULL,
    "assignee_id" varchar(100),
    "enabled" boolean NOT NULL DEFAULT TRUE,
    "created_by_id" varchar(100),
    "match_count" bigint NOT NULL DEFAULT 0,
    "last_matched_at" bigint,
    "last_message_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_triage_rules" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_triage_rules_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_rules_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_rules_mailbox" FOREIGN KEY ("mailbox_id", "business_unit_id", "organization_id") REFERENCES "inbound_mailboxes"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_triage_rules_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other')),
    CONSTRAINT "ck_triage_rules_sender_scope" CHECK ("sender_scope" IN ('Anyone', 'AnyCustomer', 'AnyCarrier', 'Domain', 'Address')),
    CONSTRAINT "ck_triage_rules_action" CHECK ("action" IN ('FileOnShipment', 'FileAndQueuePay', 'DraftReply', 'DraftLoad', 'LinkBySender', 'Ignore', 'Assign'))
);

CREATE INDEX IF NOT EXISTS "idx_triage_rules_lookup" ON "triage_rules"("organization_id", "business_unit_id", "enabled", "classification");

CREATE UNIQUE INDEX IF NOT EXISTS "uq_triage_rules_sentence" ON "triage_rules"("organization_id", "business_unit_id", "classification", "sender_scope", COALESCE("sender", ''), COALESCE("mailbox_id", ''), "action");

CREATE TABLE IF NOT EXISTS "triage_reply_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "body" text NOT NULL,
    "created_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_triage_reply_templates" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_triage_reply_templates_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_reply_templates_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_triage_reply_templates_name" ON "triage_reply_templates"("organization_id", "business_unit_id", lower("name"));

CREATE TABLE IF NOT EXISTS "triage_response_targets"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "classification" varchar(30) NOT NULL,
    "minutes" integer NOT NULL,
    "updated_by_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_triage_response_targets" PRIMARY KEY ("organization_id", "business_unit_id", "classification"),
    CONSTRAINT "fk_triage_response_targets_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_response_targets_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_triage_response_targets_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other')),
    CONSTRAINT "ck_triage_response_targets_minutes" CHECK ("minutes" BETWEEN 5 AND 43200)
);

CREATE TABLE IF NOT EXISTS "triage_settings"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "on_call_user_id" varchar(100),
    "end_of_day_minute" integer NOT NULL DEFAULT 1020,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_triage_settings" PRIMARY KEY ("organization_id", "business_unit_id"),
    CONSTRAINT "fk_triage_settings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_settings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_triage_settings_end_of_day" CHECK ("end_of_day_minute" BETWEEN 0 AND 1439)
);

CREATE TABLE IF NOT EXISTS "triage_views"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "name" varchar(80) NOT NULL,
    "query" varchar(200) NOT NULL,
    "position" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_triage_views" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_triage_views_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_triage_views_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_triage_views_user" ON "triage_views"("organization_id", "business_unit_id", "user_id", "position");

CREATE TABLE IF NOT EXISTS "inbound_sender_hints"(
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "sender_domain" varchar(255) NOT NULL,
    "classification" varchar(30) NOT NULL,
    "corrections" integer NOT NULL DEFAULT 0,
    "last_corrected_at" bigint NOT NULL,
    CONSTRAINT "pk_inbound_sender_hints" PRIMARY KEY ("organization_id", "business_unit_id", "sender_domain", "classification"),
    CONSTRAINT "fk_inbound_sender_hints_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_inbound_sender_hints_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_inbound_sender_hints_classification" CHECK ("classification" IN ('Tender', 'RateConfirmation', 'ProofOfDelivery', 'Invoice', 'StatusRequest', 'DetentionDispute', 'Other'))
);

COMMENT ON COLUMN "inbound_messages"."thread_key" IS 'The conversation a message belongs to, derived at ingest from Message-ID, In-Reply-To and References';

COMMENT ON COLUMN "inbound_messages"."due_at" IS 'When the organization''s response target for this kind of message runs out';

COMMENT ON TABLE "inbound_replies" IS 'Replies sent from a monitored address, with what the provider said about each';

COMMENT ON TABLE "triage_rules" IS 'What Nova may do on its own: which messages, from whom, on which source, and the action';

SELECT trenova_rls.reconcile();
