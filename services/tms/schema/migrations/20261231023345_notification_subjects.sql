-- A notification is about one subject and there is one open row for it per person. The row says
-- why the person got it (it was theirs, or it went to everyone who can act), how many times the
-- subject came up, when it should show (now, or later when the person set it aside), and how it
-- ended when the work it pointed at was finished.
ALTER TABLE "notifications"
    ADD COLUMN "subject_key" varchar(255) NOT NULL DEFAULT '',
    ADD COLUMN "reason" varchar(20) NOT NULL DEFAULT 'direct',
    ADD COLUMN "occurrence_count" int NOT NULL DEFAULT 1,
    ADD COLUMN "surfaces_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    ADD COLUMN "resolved_at" bigint,
    ADD COLUMN "resolution" text NOT NULL DEFAULT '',
    ADD CONSTRAINT "chk_notifications_reason" CHECK ("reason" IN ('direct', 'team')),
    ADD CONSTRAINT "chk_notifications_occurrence_count" CHECK ("occurrence_count" >= 1);

UPDATE "notifications"
SET "surfaces_at" = "created_at";

-- What went to everyone who could act, rather than to one person because it was theirs.
UPDATE "notifications"
SET "reason" = 'team'
WHERE "target_user_id" IS NULL
   OR "event_type" IN (
        'watchtower.critical',
        'agent.proposals_pending',
        'agent.proposals_reminder',
        'agent.quality_regression',
        'billing_transfer_run_completed',
        'capture.retention_reminder'
    )
   OR "event_type" LIKE 'audit.alert.%';

-- An exception that is someone's own is one subject however often it was assigned, commented on
-- or escalated.
UPDATE "notifications"
SET "subject_key" = 'watchtower:item:' || ("data" ->> 'itemId')
WHERE "event_type" IN ('watchtower.assigned', 'watchtower.commented', 'watchtower.escalated')
  AND coalesce("data" ->> 'itemId', '') <> '';

-- A critical exception told to a team is counted with the others of its kind that day.
UPDATE "notifications"
SET "subject_key" = 'watchtower:critical:' || coalesce("data" ->> 'sourceKind', 'Other') || ':'
        || to_char(to_timestamp("created_at") AT TIME ZONE 'UTC', 'YYYY-MM-DD'),
    "data" = "data" || jsonb_build_object(
        'recent', jsonb_build_array(jsonb_build_object(
            'label', "title",
            'link', coalesce("data" ->> 'link', ''),
            'at', "created_at"
        )),
        'link', '/desk/needs'
    ),
    "message" = 'Latest: ' || "title",
    "title" = initcap(substr(regexp_replace(coalesce("data" ->> 'sourceKind', 'Exception'), '([a-z])([A-Z])', '\1 \2', 'g'), 1, 1))
        || lower(substr(regexp_replace(coalesce("data" ->> 'sourceKind', 'Exception'), '([a-z])([A-Z])', '\1 \2', 'g'), 2)),
    "expires_at" = "created_at" + 259200
WHERE "event_type" = 'watchtower.critical';

-- An agent's waiting changes are one subject per agent, not one per run and again per reminder.
UPDATE "notifications"
SET "subject_key" = 'agent:proposals:' || ("data" ->> 'agentId'),
    "title" = coalesce(nullif("data" ->> 'agentName', ''), 'An agent') || ' has changes waiting for a decision',
    "data" = "data" || jsonb_build_object('link', '/desk/needs?lens=decisions')
WHERE "event_type" IN ('agent.proposals_pending', 'agent.proposals_reminder')
  AND coalesce("data" ->> 'agentId', '') <> '';

-- Keep the newest open row of each subject for each person and let it carry the count.
WITH ranked AS (
    SELECT "id",
           row_number() OVER (
               PARTITION BY "organization_id", "target_user_id", "subject_key"
               ORDER BY "created_at" DESC, "id" DESC
           ) AS "position",
           count(*) OVER (
               PARTITION BY "organization_id", "target_user_id", "subject_key"
           ) AS "total"
    FROM "notifications"
    WHERE "subject_key" <> ''
      AND "target_user_id" IS NOT NULL
      AND "dismissed_at" IS NULL
)
UPDATE "notifications" n
SET "occurrence_count" = ranked."total"
FROM ranked
WHERE n."id" = ranked."id"
  AND ranked."position" = 1;

WITH ranked AS (
    SELECT "id",
           row_number() OVER (
               PARTITION BY "organization_id", "target_user_id", "subject_key"
               ORDER BY "created_at" DESC, "id" DESC
           ) AS "position"
    FROM "notifications"
    WHERE "subject_key" <> ''
      AND "target_user_id" IS NOT NULL
      AND "dismissed_at" IS NULL
)
DELETE FROM "notifications" n
USING ranked
WHERE n."id" = ranked."id"
  AND ranked."position" > 1;
