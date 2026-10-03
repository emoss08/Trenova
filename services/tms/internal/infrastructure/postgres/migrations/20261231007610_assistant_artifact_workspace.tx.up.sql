-- A document read off an uploaded file is an artifact of its own: its fields,
-- how sure the reading is of each, and where on the page it was found.
ALTER TABLE "assistant_artifacts"
    DROP CONSTRAINT IF EXISTS "ck_assistant_artifacts_kind";

--bun:split
ALTER TABLE "assistant_artifacts"
    ADD CONSTRAINT "ck_assistant_artifacts_kind" CHECK (
        "kind" IN (
            'report_preview', 'report_run', 'email_draft', 'plan', 'entity_card',
            'table_view', 'rate_explanation', 'dashboard_ref', 'briefing',
            'inbound_message', 'run_diff', 'document', 'navigation', 'draft_edit',
            'decision_request', 'extraction'
        )
    );

--bun:split
-- A link names an artifact by a slug rather than an id, so it reads as what
-- it points to and stays the same as later versions of it arrive.
ALTER TABLE "assistant_artifacts"
    ADD COLUMN IF NOT EXISTS "slug" VARCHAR(80) NOT NULL DEFAULT '';

--bun:split
-- Existing lineages get their slug from their first version's title, numbered
-- where two in one conversation share a title; later versions take the slug
-- of the lineage they belong to.
WITH "roots" AS (
    SELECT
        "id",
        "thread_id",
        COALESCE(NULLIF(TRIM(BOTH '-' FROM LEFT(REGEXP_REPLACE(LOWER("title"), '[^a-z0-9]+', '-', 'g'), 60)), ''), 'artifact') AS "base",
        "created_at"
    FROM "assistant_artifacts"
    WHERE "lineage_id" IS NULL
),
"numbered" AS (
    SELECT
        "id",
        "base",
        ROW_NUMBER() OVER (PARTITION BY "thread_id", "base" ORDER BY "created_at", "id") AS "n"
    FROM "roots"
)
UPDATE "assistant_artifacts" AS "a"
SET "slug" = CASE WHEN "numbered"."n" = 1 THEN "numbered"."base" ELSE "numbered"."base" || '-' || "numbered"."n" END
FROM "numbered"
WHERE "a"."id" = "numbered"."id" AND "a"."slug" = '';

--bun:split
UPDATE "assistant_artifacts" AS "v"
SET "slug" = "r"."slug"
FROM "assistant_artifacts" AS "r"
WHERE "v"."lineage_id" = "r"."id"
    AND "v"."organization_id" = "r"."organization_id"
    AND "v"."business_unit_id" = "r"."business_unit_id"
    AND "v"."slug" = '';

--bun:split
-- One lineage per slug in a conversation; versions share their root's slug.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_artifacts_slug"
    ON "assistant_artifacts" ("thread_id", "slug")
    WHERE "lineage_id" IS NULL AND "slug" <> '';

--bun:split
-- The pane pages a conversation's artifacts by lineage and reads every
-- version of the lineages on a page in one go.
CREATE INDEX IF NOT EXISTS "idx_assistant_artifacts_lineage_root"
    ON "assistant_artifacts" ("organization_id", "business_unit_id", "thread_id", (COALESCE("lineage_id", "id")), "created_at" DESC);

--bun:split
COMMENT ON COLUMN "assistant_artifacts"."slug" IS 'The lineage''s name in a link (desk/c/{conversation}/a/{slug}); the same for every version and unique within the conversation';
