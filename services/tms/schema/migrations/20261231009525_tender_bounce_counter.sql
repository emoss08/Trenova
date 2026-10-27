--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "tenders"
    ADD COLUMN IF NOT EXISTS "bounce_count" smallint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "last_bounced_at" bigint,
    ADD COLUMN IF NOT EXISTS "escalated_from_tender_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "escalation_round" smallint NOT NULL DEFAULT 0;

ALTER TABLE "tenders"
    ADD CONSTRAINT "ck_tenders_bounce_count" CHECK ("bounce_count" BETWEEN 0 AND 100),
    ADD CONSTRAINT "ck_tenders_escalation_round" CHECK ("escalation_round" BETWEEN 0 AND 5),
    ADD CONSTRAINT "ck_tenders_escalated_from" CHECK (
        ("escalation_round" = 0) = ("escalated_from_tender_id" IS NULL)
    );

CREATE INDEX IF NOT EXISTS "idx_tenders_escalated_from" ON "tenders"("escalated_from_tender_id", "organization_id", "business_unit_id")
WHERE
    "escalated_from_tender_id" IS NOT NULL;

ALTER TABLE "tender_offers"
    ADD COLUMN IF NOT EXISTS "counter_rate" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "counter_note" text,
    ADD COLUMN IF NOT EXISTS "countered_at" bigint,
    ADD COLUMN IF NOT EXISTS "counter_margin_percent" numeric(9, 4),
    ADD COLUMN IF NOT EXISTS "counter_needs_approval" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "counter_decided_at" bigint,
    ADD COLUMN IF NOT EXISTS "counter_decided_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "revoked_at" bigint,
    ADD COLUMN IF NOT EXISTS "revoke_reason" text,
    ADD COLUMN IF NOT EXISTS "revoked_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "reoffered_from_id" varchar(100);

ALTER TABLE "tender_offers"
    ADD CONSTRAINT "ck_tender_offers_counter_rate" CHECK ("counter_rate" IS NULL OR "counter_rate" > 0),
    ADD CONSTRAINT "ck_tender_offers_countered" CHECK (
        "status" NOT IN ('Countered', 'CounterRejected')
        OR ("counter_rate" IS NOT NULL AND "countered_at" IS NOT NULL)
    ),
    ADD CONSTRAINT "ck_tender_offers_revoked" CHECK (
        ("status" = 'Revoked') = ("revoked_at" IS NOT NULL)
    ),
    ADD CONSTRAINT "fk_tender_offers_counter_decided_by" FOREIGN KEY ("counter_decided_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    ADD CONSTRAINT "fk_tender_offers_revoked_by" FOREIGN KEY ("revoked_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

DROP INDEX IF EXISTS "idx_tender_offers_sent_expiry";

CREATE INDEX IF NOT EXISTS "idx_tender_offers_open_expiry" ON "tender_offers"("expires_at")
WHERE
    "status" IN ('Sent', 'Countered');

ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "tender_counter_offers_enabled" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "tender_counter_margin_floor_percent" numeric(7, 4) NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS "tender_auto_accept_counters" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "tender_counter_decision_minutes" integer NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS "tender_exhausted_action" varchar(50) NOT NULL DEFAULT 'Notify',
    ADD COLUMN IF NOT EXISTS "tender_escalation_rate_increase_percent" numeric(7, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "tender_escalation_offer_ttl_minutes" integer NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS "tender_max_escalations" smallint NOT NULL DEFAULT 1;

ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "ck_dispatch_controls_tender_counter" CHECK (
        "tender_counter_margin_floor_percent" BETWEEN -100 AND 100
        AND "tender_counter_decision_minutes" BETWEEN 5 AND 1440
    ),
    ADD CONSTRAINT "ck_dispatch_controls_tender_escalation" CHECK (
        "tender_exhausted_action" IN ('Notify', 'Rebroadcast', 'Rewaterfall')
        AND "tender_escalation_rate_increase_percent" BETWEEN 0 AND 100
        AND "tender_escalation_offer_ttl_minutes" BETWEEN 5 AND 10080
        AND "tender_max_escalations" BETWEEN 0 AND 5
    );

SELECT trenova_rls.reconcile();
