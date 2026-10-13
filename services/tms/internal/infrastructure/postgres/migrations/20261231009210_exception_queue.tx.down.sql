--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "fk_dispatch_controls_exception_escalation_contact",
    DROP CONSTRAINT IF EXISTS "fk_dispatch_controls_exception_default_assignee",
    DROP CONSTRAINT IF EXISTS "ck_dispatch_controls_exception_sources",
    DROP CONSTRAINT IF EXISTS "ck_dispatch_controls_exception_escalation",
    DROP CONSTRAINT IF EXISTS "ck_dispatch_controls_exception_due";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "exception_pod_grace_hours",
    DROP COLUMN IF EXISTS "exception_missed_stop_grace_minutes",
    DROP COLUMN IF EXISTS "exception_escalation_contact_id",
    DROP COLUMN IF EXISTS "exception_default_assignee_id",
    DROP COLUMN IF EXISTS "exception_max_escalations",
    DROP COLUMN IF EXISTS "exception_escalation_minutes",
    DROP COLUMN IF EXISTS "exception_due_critical_minutes",
    DROP COLUMN IF EXISTS "exception_due_warning_minutes",
    DROP COLUMN IF EXISTS "exception_due_info_minutes";

--bun:split
DROP TABLE IF EXISTS "watchtower_item_events";

--bun:split
DROP FUNCTION IF EXISTS trenova_reject_watchtower_item_event_change();

--bun:split
DROP INDEX IF EXISTS "idx_watchtower_items_snoozed";

--bun:split
DROP INDEX IF EXISTS "idx_watchtower_items_due";

--bun:split
DROP INDEX IF EXISTS "idx_watchtower_items_assignee";

--bun:split
ALTER TABLE "watchtower_items"
    DROP CONSTRAINT IF EXISTS "fk_watchtower_items_resolved_by",
    DROP CONSTRAINT IF EXISTS "fk_watchtower_items_assignee",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_escalation",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_snooze",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_resolution_code",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_resolution",
    DROP CONSTRAINT IF EXISTS "ck_watchtower_items_status";

--bun:split
ALTER TABLE "watchtower_items"
    DROP COLUMN IF EXISTS "last_escalated_at",
    DROP COLUMN IF EXISTS "escalation_level",
    DROP COLUMN IF EXISTS "source_cleared_at",
    DROP COLUMN IF EXISTS "resolved_by_id",
    DROP COLUMN IF EXISTS "resolution_note",
    DROP COLUMN IF EXISTS "resolution_code",
    DROP COLUMN IF EXISTS "acknowledged_at",
    DROP COLUMN IF EXISTS "due_at",
    DROP COLUMN IF EXISTS "snoozed_until",
    DROP COLUMN IF EXISTS "assignee_id",
    DROP COLUMN IF EXISTS "status";
