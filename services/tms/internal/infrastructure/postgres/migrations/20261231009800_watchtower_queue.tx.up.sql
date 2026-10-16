--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- The watchtower is worked one item at a time, soonest deadline first. An
-- item says what is at stake, the record it is about, the facts that matter
-- and the events that led to it; a deadline the source reports (a dock that
-- closes, a certificate that lapses) is the due time, labelled, and only a
-- person moves it. An item handed to an agent keeps the agent and its run, so
-- the queue can show how far along the agent is.
ALTER TABLE "watchtower_items"
    ADD COLUMN IF NOT EXISTS "due_label" varchar(120),
    ADD COLUMN IF NOT EXISTS "due_source" varchar(20),
    ADD COLUMN IF NOT EXISTS "record_kind" varchar(60),
    ADD COLUMN IF NOT EXISTS "record_label" varchar(200),
    ADD COLUMN IF NOT EXISTS "facts" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "impact" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "trail" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "agent_definition_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "agent_run_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "agent_handed_at" bigint,
    ADD COLUMN IF NOT EXISTS "agent_handed_by_id" varchar(100);

--bun:split
-- Every due time set so far came from the severity policy or from a person
-- moving it; the events say which.
UPDATE "watchtower_items" AS wti
SET "due_source" = CASE
        WHEN EXISTS (
            SELECT 1
            FROM "watchtower_item_events" AS wtie
            WHERE wtie."organization_id" = wti."organization_id"
              AND wtie."business_unit_id" = wti."business_unit_id"
              AND wtie."item_id" = wti."id"
              AND wtie."kind" = 'DueChanged'
              AND wtie."actor_id" IS NOT NULL
        ) THEN 'Person'
        ELSE 'Policy'
    END
WHERE wti."due_at" IS NOT NULL;

--bun:split
ALTER TABLE "watchtower_items"
    ADD CONSTRAINT "ck_watchtower_items_due_source" CHECK ("due_source" IS NULL OR "due_source" IN ('Policy', 'Source', 'Person')),
    ADD CONSTRAINT "ck_watchtower_items_due_label" CHECK ("due_label" IS NULL OR "due_source" = 'Source'),
    ADD CONSTRAINT "ck_watchtower_items_facts" CHECK (jsonb_typeof("facts") = 'array' AND jsonb_array_length("facts") <= 6),
    ADD CONSTRAINT "ck_watchtower_items_impact" CHECK (jsonb_typeof("impact") = 'array' AND jsonb_array_length("impact") <= 6),
    ADD CONSTRAINT "ck_watchtower_items_trail" CHECK (jsonb_typeof("trail") = 'array' AND jsonb_array_length("trail") <= 8),
    ADD CONSTRAINT "ck_watchtower_items_agent" CHECK (("agent_run_id" IS NULL) = ("agent_handed_at" IS NULL)),
    ADD CONSTRAINT "fk_watchtower_items_agent_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL ("agent_definition_id"),
    ADD CONSTRAINT "fk_watchtower_items_agent_handed_by" FOREIGN KEY ("agent_handed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
-- The queue reads open, awake items soonest deadline first, undated ones
-- last, and pages by the same key.
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_queue"
    ON "watchtower_items"("organization_id", "business_unit_id", (COALESCE("due_at", 9000000000000)), "id")
    WHERE "resolved_at" IS NULL AND "status" <> 'Snoozed';

--bun:split
-- A run moving finds the items it is working.
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_agent_run"
    ON "watchtower_items"("organization_id", "business_unit_id", "agent_run_id")
    WHERE "agent_run_id" IS NOT NULL;

--bun:split
-- What a person cleared today: the items they resolved.
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_resolved_by"
    ON "watchtower_items"("organization_id", "business_unit_id", "resolved_by_id", "resolved_at")
    WHERE "resolved_by_id" IS NOT NULL;

--bun:split
-- ...and the ones they snoozed, read from the history.
CREATE INDEX IF NOT EXISTS "idx_watchtower_item_events_actor"
    ON "watchtower_item_events"("organization_id", "business_unit_id", "actor_id", "kind", "created_at")
    WHERE "actor_id" IS NOT NULL;

--bun:split
ALTER TABLE "watchtower_item_events"
    DROP CONSTRAINT IF EXISTS "ck_watchtower_item_events_kind";

--bun:split
ALTER TABLE "watchtower_item_events"
    ADD CONSTRAINT "ck_watchtower_item_events_kind" CHECK ("kind" IN (
        'Created', 'Assigned', 'Unassigned', 'Acknowledged', 'Started', 'Snoozed', 'Woke',
        'DueChanged', 'Commented', 'Escalated', 'Resolved', 'Reopened', 'SourceResolved',
        'SourceReopened', 'HandedOff'
    ));

--bun:split
-- Which items each reader has opened, so the queue marks only the ones they
-- have not.
CREATE TABLE IF NOT EXISTS "watchtower_item_reads"(
    "item_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "seen_at" bigint NOT NULL,
    CONSTRAINT "pk_watchtower_item_reads" PRIMARY KEY ("item_id", "user_id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_watchtower_item_reads_item" FOREIGN KEY ("item_id", "business_unit_id", "organization_id") REFERENCES "watchtower_items"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_item_reads_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_item_reads_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_item_reads_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_watchtower_item_reads_seen_at" CHECK ("seen_at" > 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_watchtower_item_reads_user"
    ON "watchtower_item_reads"("organization_id", "business_unit_id", "user_id", "item_id");

--bun:split
SELECT trenova_rls.reconcile();
