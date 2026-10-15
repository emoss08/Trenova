--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "ck_dispatch_controls_tender_escalation",
    DROP CONSTRAINT IF EXISTS "ck_dispatch_controls_tender_counter",
    DROP COLUMN IF EXISTS "tender_max_escalations",
    DROP COLUMN IF EXISTS "tender_escalation_offer_ttl_minutes",
    DROP COLUMN IF EXISTS "tender_escalation_rate_increase_percent",
    DROP COLUMN IF EXISTS "tender_exhausted_action",
    DROP COLUMN IF EXISTS "tender_counter_decision_minutes",
    DROP COLUMN IF EXISTS "tender_auto_accept_counters",
    DROP COLUMN IF EXISTS "tender_counter_margin_floor_percent",
    DROP COLUMN IF EXISTS "tender_counter_offers_enabled";

--bun:split
DROP INDEX IF EXISTS "idx_tender_offers_open_expiry";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_tender_offers_sent_expiry" ON "tender_offers"("expires_at")
WHERE
    "status" = 'Sent';

--bun:split
ALTER TABLE "tender_offers"
    DROP CONSTRAINT IF EXISTS "fk_tender_offers_revoked_by",
    DROP CONSTRAINT IF EXISTS "fk_tender_offers_counter_decided_by",
    DROP CONSTRAINT IF EXISTS "ck_tender_offers_revoked",
    DROP CONSTRAINT IF EXISTS "ck_tender_offers_countered",
    DROP CONSTRAINT IF EXISTS "ck_tender_offers_counter_rate",
    DROP COLUMN IF EXISTS "reoffered_from_id",
    DROP COLUMN IF EXISTS "revoked_by_id",
    DROP COLUMN IF EXISTS "revoke_reason",
    DROP COLUMN IF EXISTS "revoked_at",
    DROP COLUMN IF EXISTS "counter_decided_by_id",
    DROP COLUMN IF EXISTS "counter_decided_at",
    DROP COLUMN IF EXISTS "counter_needs_approval",
    DROP COLUMN IF EXISTS "counter_margin_percent",
    DROP COLUMN IF EXISTS "countered_at",
    DROP COLUMN IF EXISTS "counter_note",
    DROP COLUMN IF EXISTS "counter_rate";

--bun:split
DROP INDEX IF EXISTS "idx_tenders_escalated_from";

--bun:split
ALTER TABLE "tenders"
    DROP CONSTRAINT IF EXISTS "ck_tenders_escalated_from",
    DROP CONSTRAINT IF EXISTS "ck_tenders_escalation_round",
    DROP CONSTRAINT IF EXISTS "ck_tenders_bounce_count",
    DROP COLUMN IF EXISTS "escalation_round",
    DROP COLUMN IF EXISTS "escalated_from_tender_id",
    DROP COLUMN IF EXISTS "last_bounced_at",
    DROP COLUMN IF EXISTS "bounce_count";
