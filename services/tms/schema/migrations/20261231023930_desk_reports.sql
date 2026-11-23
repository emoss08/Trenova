-- The people of an organization as reports name them. users carries only a business unit, so a
-- report that joined it directly could reach a person of another organization in the same
-- business unit, and keel refuses to join tables whose tenant columns differ. A membership
-- carries both, so the view is keyed by the organization and joins only its members; it reads
-- through the caller's own row-level security (security_invoker).
CREATE OR REPLACE VIEW "organization_people" WITH (security_invoker = true) AS
SELECT
    m."user_id" AS "id",
    m."business_unit_id",
    m."organization_id",
    u."name",
    NULLIF(u."job_title", '') AS "job_title",
    u."status",
    m."joined_at"
FROM "user_organization_memberships" m
JOIN "users" u ON u."id" = m."user_id" AND u."business_unit_id" = m."business_unit_id";

COMMENT ON VIEW "organization_people" IS 'Members of each organization by name, job title and account status, for reports';

-- One row each time a watchtower item is resolved. Items are removed 30 days after they resolve,
-- so handled work and resolution reasons are reported from here. Rows are written in the
-- transaction that resolves the item, keyed by the item's version after that write, so a retried
-- write records nothing twice and a reopened item resolved again records a second row. Nothing
-- an item reads (title, summary, facts) is kept.
CREATE TABLE IF NOT EXISTS "watchtower_resolutions" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "item_id" varchar(100) NOT NULL,
    "item_version" bigint NOT NULL,
    "case_id" varchar(100) NOT NULL,
    "kind" varchar(50) NOT NULL,
    "severity" varchar(20) NOT NULL,
    "resolved_via" varchar(20) NOT NULL,
    "resolution_code" varchar(30) NOT NULL,
    "assignee_id" varchar(100),
    "resolved_by_id" varchar(100),
    "occurred_at" bigint NOT NULL,
    "acknowledged_at" bigint,
    "due_at" bigint,
    "resolved_at" bigint NOT NULL,
    "seconds_to_acknowledge" bigint,
    "seconds_to_resolve" bigint NOT NULL,
    "past_due" boolean NOT NULL DEFAULT FALSE,
    "escalation_level" smallint NOT NULL DEFAULT 0,
    "agent_handed" boolean NOT NULL DEFAULT FALSE,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_watchtower_resolutions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_watchtower_resolutions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_resolutions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_watchtower_resolutions_severity" CHECK ("severity" IN ('Info', 'Warning', 'Critical')),
    CONSTRAINT "ck_watchtower_resolutions_resolved_via" CHECK ("resolved_via" IN ('Person', 'Agent', 'SourceCleared')),
    CONSTRAINT "ck_watchtower_resolutions_resolution_code" CHECK ("resolution_code" IN (
        'Fixed', 'CarrierDelay', 'ShipperDelay', 'ReceiverDelay', 'CustomerChange',
        'Rescheduled', 'Weather', 'Equipment', 'DriverIssue', 'DocumentReceived',
        'DataCorrected', 'Duplicate', 'NoActionNeeded', 'Other', 'Cleared'
    )),
    CONSTRAINT "ck_watchtower_resolutions_resolver" CHECK ("resolved_via" <> 'SourceCleared' OR "resolved_by_id" IS NULL),
    CONSTRAINT "ck_watchtower_resolutions_durations" CHECK (
        "seconds_to_resolve" >= 0
        AND ("seconds_to_acknowledge" IS NULL OR "seconds_to_acknowledge" >= 0)
        AND "escalation_level" >= 0
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_watchtower_resolutions_item_version"
    ON "watchtower_resolutions" ("organization_id", "business_unit_id", "item_id", "item_version");

-- Every report over resolutions reads a window of resolution times.
CREATE INDEX IF NOT EXISTS "idx_watchtower_resolutions_tenant_resolved"
    ON "watchtower_resolutions" ("organization_id", "business_unit_id", "resolved_at");

-- Retention removes the oldest rows across every organization.
CREATE INDEX IF NOT EXISTS "idx_watchtower_resolutions_resolved"
    ON "watchtower_resolutions" ("resolved_at");

CREATE OR REPLACE FUNCTION trenova_reject_watchtower_resolution_change() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'watchtower_resolutions is append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS "trg_watchtower_resolutions_append_only" ON "watchtower_resolutions";

CREATE TRIGGER "trg_watchtower_resolutions_append_only"
    BEFORE UPDATE ON "watchtower_resolutions"
    FOR EACH ROW EXECUTE FUNCTION trenova_reject_watchtower_resolution_change();

COMMENT ON TABLE "watchtower_resolutions" IS 'One row each time a watchtower item is resolved: who, why, how long it took; kept 25 months';

INSERT INTO "watchtower_resolutions" (
    "id", "business_unit_id", "organization_id", "item_id", "item_version", "case_id", "kind",
    "severity", "resolved_via", "resolution_code", "assignee_id", "resolved_by_id", "occurred_at",
    "acknowledged_at", "due_at", "resolved_at", "seconds_to_acknowledge", "seconds_to_resolve",
    "past_due", "escalation_level", "agent_handed", "created_at"
)
SELECT
    'wtr_' || substr(md5(i."id" || ':' || i."version"::text), 1, 26),
    i."business_unit_id",
    i."organization_id",
    i."id",
    i."version",
    COALESCE(i."about_id", i."id"),
    i."source_kind",
    i."severity",
    CASE WHEN i."resolved_by_id" IS NULL THEN 'SourceCleared' ELSE 'Person' END,
    COALESCE(i."resolution_code", 'Cleared'),
    i."assignee_id",
    i."resolved_by_id",
    i."occurred_at",
    i."acknowledged_at",
    i."due_at",
    i."resolved_at",
    CASE WHEN i."acknowledged_at" IS NULL THEN NULL ELSE GREATEST(i."acknowledged_at" - i."occurred_at", 0) END,
    GREATEST(i."resolved_at" - i."occurred_at", 0),
    i."due_at" IS NOT NULL AND i."resolved_at" > i."due_at",
    i."escalation_level",
    i."agent_handed_at" IS NOT NULL,
    i."resolved_at"
FROM "watchtower_items" i
WHERE i."resolved_at" IS NOT NULL
ON CONFLICT DO NOTHING;

-- Model calls added up by the organization's local day, person, feature, surface, model and
-- provider protocol. ai-analytics-refresh recomputes every local day a refresh window touches
-- from the calls on it, so the rollup never scans more than the days being folded and a report
-- over a year reads a row per person per feature per day. Input is stored uncached, with the
-- cached part of the OpenAI protocols taken out, so totals mean the same on every provider.
-- The id is derived from the key, which makes the fold an idempotent upsert.
CREATE TABLE IF NOT EXISTS "ai_usage_daily" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "usage_date" bigint NOT NULL,
    "user_id" varchar(100),
    "feature" varchar(50),
    "surface" varchar(50) NOT NULL,
    "model" varchar(200) NOT NULL,
    "provider_kind" varchar(50) NOT NULL,
    "calls" bigint NOT NULL DEFAULT 0,
    "failed_calls" bigint NOT NULL DEFAULT 0,
    "unpriced_calls" bigint NOT NULL DEFAULT 0,
    "input_tokens" bigint NOT NULL DEFAULT 0,
    "cache_read_tokens" bigint NOT NULL DEFAULT 0,
    "cache_write_tokens" bigint NOT NULL DEFAULT 0,
    "output_tokens" bigint NOT NULL DEFAULT 0,
    "reasoning_tokens" bigint NOT NULL DEFAULT 0,
    "advisor_tokens" bigint NOT NULL DEFAULT 0,
    "total_tokens" bigint NOT NULL DEFAULT 0,
    "cost_usd" numeric(19, 6) NOT NULL DEFAULT 0,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_ai_usage_daily" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_ai_usage_daily_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_usage_daily_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_ai_usage_daily_counts" CHECK (
        "calls" >= 0 AND "failed_calls" >= 0 AND "unpriced_calls" >= 0
        AND "failed_calls" <= "calls" AND "unpriced_calls" <= "calls"
    )
);

-- Reports read a window of days; the refresh prunes the oldest days of a tenant.
CREATE INDEX IF NOT EXISTS "idx_ai_usage_daily_tenant_date"
    ON "ai_usage_daily" ("organization_id", "business_unit_id", "usage_date");

COMMENT ON TABLE "ai_usage_daily" IS 'Model calls by organization-local day, person, feature, surface and model: tokens, estimated cost, unpriced calls; kept 25 months';

INSERT INTO "ai_usage_daily" (
    "id", "business_unit_id", "organization_id", "usage_date", "user_id", "feature", "surface",
    "model", "provider_kind", "calls", "failed_calls", "unpriced_calls", "input_tokens",
    "cache_read_tokens", "cache_write_tokens", "output_tokens", "reasoning_tokens",
    "advisor_tokens", "total_tokens", "cost_usd", "updated_at"
)
SELECT
    'aiud_' || md5(
        CAST(c."usage_date" AS text) || '|' || COALESCE(c."user_id", '') || '|'
        || COALESCE(c."feature", '') || '|' || c."surface" || '|' || c."model" || '|'
        || c."provider_kind"
    ),
    c."business_unit_id",
    c."organization_id",
    c."usage_date",
    c."user_id",
    c."feature",
    c."surface",
    c."model",
    c."provider_kind",
    count(*),
    sum(c."failed"),
    sum(c."unpriced"),
    sum(c."input_tokens"),
    sum(c."cache_read_tokens"),
    sum(c."cache_write_tokens"),
    sum(c."output_tokens"),
    sum(c."reasoning_tokens"),
    sum(c."advisor_tokens"),
    sum(c."input_tokens" + c."cache_read_tokens" + c."cache_write_tokens" + c."output_tokens" + c."advisor_tokens"),
    COALESCE(sum(c."cost_usd"), 0),
    EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
FROM (
    SELECT
        r."business_unit_id",
        r."organization_id",
        CAST(extract(epoch FROM timezone(z."zone", date_trunc('day', timezone(z."zone", to_timestamp(r."created_at"))))) AS bigint) AS "usage_date",
        r."user_id",
        r."feature",
        r."surface",
        r."model",
        r."provider_kind",
        CASE WHEN r."succeeded" IS FALSE THEN 1 ELSE 0 END AS "failed",
        CASE WHEN r."cost_usd" IS NULL THEN 1 ELSE 0 END AS "unpriced",
        CASE
            WHEN r."provider_kind" = 'AnthropicMessages' THEN r."input_tokens"::bigint
            ELSE GREATEST(r."input_tokens"::bigint - r."cache_read_tokens" - r."cache_write_tokens", 0)
        END AS "input_tokens",
        r."cache_read_tokens"::bigint AS "cache_read_tokens",
        r."cache_write_tokens"::bigint AS "cache_write_tokens",
        r."output_tokens"::bigint AS "output_tokens",
        r."reasoning_tokens"::bigint AS "reasoning_tokens",
        (r."advisor_input_tokens" + r."advisor_cache_read_tokens" + r."advisor_cache_write_tokens" + r."advisor_output_tokens")::bigint AS "advisor_tokens",
        r."cost_usd"
    FROM "ai_usage_records" r
    CROSS JOIN LATERAL (
        SELECT COALESCE((SELECT o."timezone" FROM "organizations" o WHERE o."id" = r."organization_id"), 'UTC') AS "zone"
    ) z
) c
GROUP BY
    c."business_unit_id", c."organization_id", c."usage_date", c."user_id", c."feature",
    c."surface", c."model", c."provider_kind"
ON CONFLICT DO NOTHING;
