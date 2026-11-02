-- Nova runs on one provider an organization chooses for it, always with Nova's own model
-- (Claude Haiku 5.5), instead of on whichever providers were assigned each of its tasks. The
-- Desk keeps routing by assignment, since there an organization picks its model.
--
-- Nova calls Claude by its Claude API id, so its provider speaks the Anthropic Messages API on
-- a platform that names models that way: Bedrock prefixes its own ids and Foundry calls a
-- deployment the organization named.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "nova" boolean NOT NULL DEFAULT FALSE;

ALTER TABLE "ai_providers"
    DROP CONSTRAINT IF EXISTS "ck_ai_providers_nova";

ALTER TABLE "ai_providers"
    ADD CONSTRAINT "ck_ai_providers_nova" CHECK (NOT "nova" OR ("kind" = 'AnthropicMessages' AND "platform" IN ('Direct', 'VertexAI', 'ClaudePlatformAWS', 'ClaudePlatformGoogleCloud')));

COMMENT ON COLUMN "ai_providers"."nova" IS 'The provider Nova runs on, with Nova''s own model; at most one per organization';

-- Each organization's Nova moves to the Anthropic provider that served the most of it: one
-- assigned a Nova task first, then an enabled one, then by routing order.
WITH "nova_tasks"("task") AS (
    VALUES ('DocumentClassification'), ('DocumentExtraction'), ('BillingDiagnosis'), ('FormulaAssistant'), ('OperationalInsights'), ('DailyBriefing'), ('QueryCompose'), ('ChapterTitling'), ('InboundClassification'), ('AccountingMapping'), ('RemittanceExtraction'), ('PaperworkReading'), ('PaletteAnswer')
),
"ranked" AS (
    SELECT
        "p"."id",
        "p"."organization_id",
        "p"."business_unit_id",
        ROW_NUMBER() OVER (PARTITION BY "p"."organization_id", "p"."business_unit_id" ORDER BY EXISTS (
                SELECT
                    1
                FROM
                    "nova_tasks" "n"
                WHERE
                    "n"."task" = ANY (COALESCE("p"."tasks", '{}'::text[]))) DESC, "p"."enabled" DESC, "p"."priority", "p"."created_at", "p"."id") AS "rank"
    FROM
        "ai_providers" "p"
    WHERE
        "p"."kind" = 'AnthropicMessages'
        AND "p"."platform" IN ('Direct', 'VertexAI', 'ClaudePlatformAWS', 'ClaudePlatformGoogleCloud'))
UPDATE
    "ai_providers" "p"
SET
    "nova" = TRUE
FROM
    "ranked" "r"
WHERE
    "r"."rank" = 1
    AND "p"."id" = "r"."id"
    AND "p"."organization_id" = "r"."organization_id"
    AND "p"."business_unit_id" = "r"."business_unit_id"
    AND NOT EXISTS (
        SELECT
            1
        FROM
            "ai_providers" "chosen"
        WHERE
            "chosen"."organization_id" = "p"."organization_id"
            AND "chosen"."business_unit_id" = "p"."business_unit_id"
            AND "chosen"."nova");

-- Nova's tasks are no longer assigned. A provider left serving nothing is turned off rather
-- than left enabled with no work, which saving it would refuse.
UPDATE
    "ai_providers"
SET
    "tasks" = ARRAY (
        SELECT
            "task"
        FROM
            UNNEST("tasks") AS "task"
        WHERE
            "task" NOT IN ('DocumentClassification', 'DocumentExtraction', 'BillingDiagnosis', 'FormulaAssistant', 'OperationalInsights', 'DailyBriefing', 'QueryCompose', 'ChapterTitling', 'InboundClassification', 'AccountingMapping', 'RemittanceExtraction', 'PaperworkReading', 'PaletteAnswer'))
WHERE
    "tasks" && ARRAY['DocumentClassification', 'DocumentExtraction', 'BillingDiagnosis', 'FormulaAssistant', 'OperationalInsights', 'DailyBriefing', 'QueryCompose', 'ChapterTitling', 'InboundClassification', 'AccountingMapping', 'RemittanceExtraction', 'PaperworkReading', 'PaletteAnswer']::text[];

UPDATE
    "ai_providers"
SET
    "enabled" = FALSE
WHERE
    "enabled"
    AND NOT "nova"
    AND COALESCE(CARDINALITY("tasks"), 0) = 0;
