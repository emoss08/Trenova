-- Insights are operational findings surfaced on the home screen.
--
-- The columns are split along the line that matters: metrics, links, severity
-- and category are what a detector computed and are facts; headline, narrative
-- and recommendation are prose a model wrote about those facts. narrated says
-- which of the two produced the wording, so an insight whose narration failed is
-- still a complete record rather than a blank card.
--
-- The enum-shaped columns are varchar with CHECK constraints rather than
-- Postgres enum types deliberately: categories and severities grow as detectors
-- are added, and ALTER TYPE ... ADD VALUE cannot run inside a transaction, so a
-- real enum would make every future detector a two-part non-transactional
-- migration for no benefit.
CREATE TABLE IF NOT EXISTS "insights"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "detector_key" varchar(100) NOT NULL,
    "category" varchar(50) NOT NULL,
    "severity" varchar(50) NOT NULL,
    "status" varchar(50) NOT NULL DEFAULT 'Active',
    "dedupe_key" varchar(255) NOT NULL,
    "subject" varchar(255),
    "headline" text NOT NULL,
    "narrative" text,
    "recommendation" text,
    "narrated" boolean NOT NULL DEFAULT FALSE,
    "metrics" jsonb NOT NULL DEFAULT '[]',
    "links" jsonb NOT NULL DEFAULT '[]',
    "window_start" bigint NOT NULL DEFAULT 0,
    "window_end" bigint NOT NULL,
    "detected_at" bigint NOT NULL,
    "stale_at" bigint NOT NULL,
    "dismissed_at" bigint,
    "dismissed_by_id" varchar(100),
    "dismiss_reason" text,
    "model_identifier" varchar(255),
    "provider_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_insights" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_insights_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_insights_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- A dismissing user who is later deleted must not take the dismissal with
    -- them: the insight stays suppressed, it just stops naming who suppressed it.
    CONSTRAINT "fk_insights_dismissed_by" FOREIGN KEY ("dismissed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_insights_category" CHECK ("category" IN ('ServiceQuality', 'CashFlow', 'CostLeakage', 'Compliance')),
    CONSTRAINT "ck_insights_severity" CHECK ("severity" IN ('Info', 'Warning', 'Critical')),
    CONSTRAINT "ck_insights_status" CHECK ("status" IN ('Active', 'Dismissed', 'Resolved', 'Superseded')),
    CONSTRAINT "ck_insights_window" CHECK ("window_start" <= "window_end"),
    -- A dismissed insight without a dismissal time cannot be aged out of
    -- suppression, so it would be suppressed forever by accident.
    CONSTRAINT "ck_insights_dismissed_at" CHECK ("status" <> 'Dismissed' OR "dismissed_at" IS NOT NULL)
);

--bun:split
-- One active insight per finding, enforced rather than assumed. A refresh
-- supersedes the previous row before inserting the new one, and this index is
-- what makes two concurrent refreshes unable to leave a duplicate card on
-- someone's home screen.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_insights_active_dedupe" ON "insights"("organization_id", "business_unit_id", "dedupe_key")
WHERE
    "status" = 'Active';

--bun:split
-- The home widget reads active insights for one tenant, newest first. Severity
-- ordering is applied in the query rather than the index because the ranks do
-- not sort alphabetically.
CREATE INDEX IF NOT EXISTS "idx_insights_active" ON "insights"("organization_id", "business_unit_id", "status", "detected_at" DESC);

--bun:split
-- Refreshing looks up recent dismissals by dedupe key to decide what to suppress.
CREATE INDEX IF NOT EXISTS "idx_insights_dismissed_dedupe" ON "insights"("organization_id", "business_unit_id", "dedupe_key", "dismissed_at" DESC)
WHERE
    "status" = 'Dismissed';

--bun:split
-- Resolving a run's leftovers filters active rows by the detector that ran.
CREATE INDEX IF NOT EXISTS "idx_insights_detector" ON "insights"("organization_id", "business_unit_id", "detector_key", "status");

COMMENT ON COLUMN "insights"."narrated" IS 'True when a model wrote the prose; false means the wording is the detector''s own deterministic text';

COMMENT ON COLUMN "insights"."metrics" IS 'Numbers the detector computed. Authoritative; never parsed back out of model output';

COMMENT ON COLUMN "insights"."links" IS 'In-application paths built by the detector to the records it counted; never model-supplied';

COMMENT ON COLUMN "insights"."dedupe_key" IS 'Identifies the same finding across refreshes, so numbers update in place and a dismissal sticks';

COMMENT ON COLUMN "insights"."stale_at" IS 'When these numbers stop being worth trusting; the client says so rather than hiding the finding';
