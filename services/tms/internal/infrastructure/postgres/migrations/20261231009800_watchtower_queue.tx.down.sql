--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "watchtower_item_reads";

--bun:split
DELETE FROM "watchtower_item_events" WHERE "kind" = 'HandedOff';

--bun:split
ALTER TABLE "watchtower_item_events"
    DROP CONSTRAINT IF EXISTS "ck_watchtower_item_events_kind";

--bun:split
ALTER TABLE "watchtower_item_events"
    ADD CONSTRAINT "ck_watchtower_item_events_kind" CHECK ("kind" IN (
        'Created', 'Assigned', 'Unassigned', 'Acknowledged', 'Started', 'Snoozed', 'Woke',
        'DueChanged', 'Commented', 'Escalated', 'Resolved', 'Reopened', 'SourceResolved',
        'SourceReopened'
    ));

--bun:split
DROP INDEX IF EXISTS "idx_watchtower_item_events_actor";

--bun:split
DROP INDEX IF EXISTS "idx_watchtower_items_resolved_by";

--bun:split
DROP INDEX IF EXISTS "idx_watchtower_items_agent_run";

--bun:split
DROP INDEX IF EXISTS "idx_watchtower_items_queue";

--bun:split
ALTER TABLE "watchtower_items"
    DROP CONSTRAINT IF EXISTS "fk_watchtower_items_agent_handed_by",
    DROP CONSTRAINT IF EXISTS "fk_watchtower_items_agent_definition",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_agent",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_trail",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_impact",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_facts",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_due_label",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_due_source";

--bun:split
ALTER TABLE "watchtower_items"
    DROP COLUMN IF EXISTS "agent_handed_by_id",
    DROP COLUMN IF EXISTS "agent_handed_at",
    DROP COLUMN IF EXISTS "agent_run_id",
    DROP COLUMN IF EXISTS "agent_definition_id",
    DROP COLUMN IF EXISTS "trail",
    DROP COLUMN IF EXISTS "impact",
    DROP COLUMN IF EXISTS "facts",
    DROP COLUMN IF EXISTS "record_label",
    DROP COLUMN IF EXISTS "record_kind",
    DROP COLUMN IF EXISTS "due_source",
    DROP COLUMN IF EXISTS "due_label";
