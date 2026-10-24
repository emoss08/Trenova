--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- What a customer sees through a tracking link and which status emails it is
-- sent. One row per customer; a customer without one gets the default view and
-- no emails.
CREATE TABLE IF NOT EXISTS "customer_tracking_profiles"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "link_enabled" boolean NOT NULL DEFAULT TRUE,
    "visible_fields" varchar(30)[] NOT NULL DEFAULT ARRAY['Status', 'Windows', 'Actuals', 'Eta', 'References']::varchar(30)[],
    "link_ttl_days" smallint NOT NULL DEFAULT 30,
    "notify_picked_up" boolean NOT NULL DEFAULT FALSE,
    "notify_delayed" boolean NOT NULL DEFAULT FALSE,
    "notify_delivered" boolean NOT NULL DEFAULT FALSE,
    "notify_pod_ready" boolean NOT NULL DEFAULT FALSE,
    "include_link" boolean NOT NULL DEFAULT TRUE,
    "recipients" text,
    "notifications_enabled_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_customer_tracking_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_customer_tracking_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_customer_tracking_profiles_ttl" CHECK ("link_ttl_days" BETWEEN 1 AND 180),
    CONSTRAINT "chk_customer_tracking_profiles_fields" CHECK ("visible_fields" <@ ARRAY['Status', 'LocationNames', 'Windows', 'Actuals', 'Eta', 'References', 'Carrier', 'Equipment', 'Freight', 'LastLocation', 'Holds', 'Notes', 'ProofOfDelivery']::varchar(30)[]),
    CONSTRAINT "chk_customer_tracking_profiles_enabled_at" CHECK (("notifications_enabled_at" IS NOT NULL) = ("notify_picked_up" OR "notify_delayed" OR "notify_delivered" OR "notify_pod_ready"))
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_customer_tracking_profiles_customer"
    ON "customer_tracking_profiles"("organization_id", "business_unit_id", "customer_id");

CREATE INDEX IF NOT EXISTS "idx_customer_tracking_profiles_notifying"
    ON "customer_tracking_profiles"("organization_id", "business_unit_id")
    WHERE "notifications_enabled_at" IS NOT NULL;

-- A shareable link to one shipment's tracking page, opened without signing in.
-- Only the token's hash is kept. A shipment may have several live links: one
-- shared by hand and one in each status email.
CREATE TABLE IF NOT EXISTS "customer_tracking_links"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "token_hash" varchar(128) NOT NULL,
    "purpose" varchar(20) NOT NULL,
    "recipient_email" varchar(320),
    "expires_at" bigint NOT NULL,
    "revoked_at" bigint,
    "last_viewed_at" bigint,
    "view_count" bigint NOT NULL DEFAULT 0,
    "created_by_id" varchar(100),
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_customer_tracking_links" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_customer_tracking_links_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_customer_tracking_links_purpose" CHECK ("purpose" IN ('Shared', 'Notification')),
    CONSTRAINT "chk_customer_tracking_links_expiry" CHECK ("expires_at" > "created_at"),
    CONSTRAINT "chk_customer_tracking_links_views" CHECK ("view_count" >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_customer_tracking_links_hash"
    ON "customer_tracking_links"("token_hash");

CREATE INDEX IF NOT EXISTS "idx_customer_tracking_links_shipment"
    ON "customer_tracking_links"("organization_id", "business_unit_id", "shipment_id", "created_at" DESC);

COMMENT ON TABLE "customer_tracking_links" IS 'Hashed customer tracking link tokens. The raw token only ever appears in the link given to the customer.';

-- The ledger of status emails a customer is owed. Its natural key makes each
-- event owe exactly one email however often it is seen, and the email is sent
-- under the row's id as its idempotency key.
CREATE TABLE IF NOT EXISTS "customer_status_notifications"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "event" varchar(20) NOT NULL,
    "subject_key" varchar(100) NOT NULL DEFAULT '',
    "stop_id" varchar(100),
    "occurred_at" bigint NOT NULL,
    "estimated_at" bigint,
    "status" varchar(20) NOT NULL,
    "skip_reason" varchar(30),
    "recipients" text[] NOT NULL DEFAULT '{}',
    "link_id" varchar(100),
    "email_message_id" varchar(100),
    "attempts" smallint NOT NULL DEFAULT 0,
    "last_error" varchar(1000),
    "next_attempt_at" bigint NOT NULL,
    "sent_at" bigint,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_customer_status_notifications" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_customer_status_notifications_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_customer_status_notifications_event" CHECK ("event" IN ('PickedUp', 'Delayed', 'Delivered', 'PodReady')),
    CONSTRAINT "chk_customer_status_notifications_status" CHECK ("status" IN ('Pending', 'Sent', 'Skipped', 'Failed')),
    CONSTRAINT "chk_customer_status_notifications_skip" CHECK (("status" = 'Skipped') = ("skip_reason" IS NOT NULL)),
    CONSTRAINT "chk_customer_status_notifications_sent" CHECK (("status" = 'Sent') = ("sent_at" IS NOT NULL)),
    CONSTRAINT "chk_customer_status_notifications_attempts" CHECK ("attempts" >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_customer_status_notifications_event"
    ON "customer_status_notifications"("organization_id", "business_unit_id", "shipment_id", "event", "subject_key");

CREATE INDEX IF NOT EXISTS "idx_customer_status_notifications_pending"
    ON "customer_status_notifications"("next_attempt_at")
    WHERE "status" = 'Pending';

CREATE INDEX IF NOT EXISTS "idx_customer_status_notifications_shipment"
    ON "customer_status_notifications"("organization_id", "business_unit_id", "shipment_id", "created_at" DESC);

SELECT trenova_rls.reconcile();
