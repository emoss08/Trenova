--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A watchtower item becomes an owned exception: someone it belongs to, where
-- it stands, when it is due, how it was resolved and how many times it has
-- escalated. A person's resolution holds until the source itself clears, so
-- source_cleared_at records that the condition behind a hand-resolved item
-- has gone, after which the source may raise it again.
ALTER TABLE "watchtower_items"
    ADD COLUMN IF NOT EXISTS "status" varchar(20) NOT NULL DEFAULT 'Open',
    ADD COLUMN IF NOT EXISTS "assignee_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "due_at" bigint,
    ADD COLUMN IF NOT EXISTS "snoozed_until" bigint,
    ADD COLUMN IF NOT EXISTS "acknowledged_at" bigint,
    ADD COLUMN IF NOT EXISTS "resolution_code" varchar(30),
    ADD COLUMN IF NOT EXISTS "resolution_note" text,
    ADD COLUMN IF NOT EXISTS "resolved_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "source_cleared_at" bigint,
    ADD COLUMN IF NOT EXISTS "escalation_level" smallint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "last_escalated_at" bigint;

--bun:split
UPDATE "watchtower_items"
SET "status" = 'Resolved',
    "resolution_code" = 'Cleared'
WHERE "resolved_at" IS NOT NULL;

--bun:split
ALTER TABLE "watchtower_items"
    ADD CONSTRAINT "ck_watchtower_items_status" CHECK ("status" IN ('Open', 'Acknowledged', 'InProgress', 'Snoozed', 'Resolved')),
    ADD CONSTRAINT "ck_watchtower_items_resolution" CHECK (("status" = 'Resolved') = ("resolved_at" IS NOT NULL)),
    ADD CONSTRAINT "ck_watchtower_items_resolution_code" CHECK (
        "resolution_code" IS NULL
        OR "resolution_code" IN (
            'Fixed', 'CarrierDelay', 'ShipperDelay', 'ReceiverDelay', 'CustomerChange',
            'Rescheduled', 'Weather', 'Equipment', 'DriverIssue', 'DocumentReceived',
            'DataCorrected', 'Duplicate', 'NoActionNeeded', 'Other', 'Cleared'
        )
    ),
    ADD CONSTRAINT "ck_watchtower_items_snooze" CHECK ("status" <> 'Snoozed' OR "snoozed_until" IS NOT NULL),
    ADD CONSTRAINT "ck_watchtower_items_escalation" CHECK ("escalation_level" >= 0),
    ADD CONSTRAINT "fk_watchtower_items_assignee" FOREIGN KEY ("assignee_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    ADD CONSTRAINT "fk_watchtower_items_resolved_by" FOREIGN KEY ("resolved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_assignee"
    ON "watchtower_items"("organization_id", "business_unit_id", "assignee_id", "due_at")
    WHERE "resolved_at" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_due"
    ON "watchtower_items"("due_at")
    WHERE "resolved_at" IS NULL AND "due_at" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_watchtower_items_snoozed"
    ON "watchtower_items"("snoozed_until")
    WHERE "status" = 'Snoozed';

--bun:split
-- Every assignment, status change, due time, comment, escalation and
-- resolution on an exception, with who did it. Append-only: the timeline is
-- the record of how the exception was handled.
CREATE TABLE IF NOT EXISTS "watchtower_item_events"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "item_id" varchar(100) NOT NULL,
    "kind" varchar(30) NOT NULL,
    "actor_id" varchar(100),
    "from_value" varchar(200),
    "to_value" varchar(200),
    "note" text,
    "created_at" bigint NOT NULL,
    CONSTRAINT "pk_watchtower_item_events" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_watchtower_item_events_item" FOREIGN KEY ("item_id", "business_unit_id", "organization_id") REFERENCES "watchtower_items"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_item_events_actor" FOREIGN KEY ("actor_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_watchtower_item_events_kind" CHECK ("kind" IN (
        'Created', 'Assigned', 'Unassigned', 'Acknowledged', 'Started', 'Snoozed', 'Woke',
        'DueChanged', 'Commented', 'Escalated', 'Resolved', 'Reopened', 'SourceResolved',
        'SourceReopened'
    )),
    CONSTRAINT "ck_watchtower_item_events_created_at" CHECK ("created_at" > 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_watchtower_item_events_item"
    ON "watchtower_item_events"("organization_id", "business_unit_id", "item_id", "created_at", "id");

--bun:split
CREATE OR REPLACE FUNCTION trenova_reject_watchtower_item_event_change() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'watchtower_item_events is append-only';
END;
$$ LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS "trg_watchtower_item_events_append_only" ON "watchtower_item_events";

--bun:split
CREATE TRIGGER "trg_watchtower_item_events_append_only"
    BEFORE UPDATE ON "watchtower_item_events"
    FOR EACH ROW EXECUTE FUNCTION trenova_reject_watchtower_item_event_change();

--bun:split
-- How long an exception has before it is overdue, by severity, how often an
-- overdue one escalates and how many times, and who takes what no rule
-- assigns and who hears about escalations. A stop is missed once its window
-- has been closed past the grace, and a delivered load is missing its POD once
-- the POD grace has passed since the delivery.
ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "exception_due_info_minutes" integer NOT NULL DEFAULT 1440,
    ADD COLUMN IF NOT EXISTS "exception_due_warning_minutes" integer NOT NULL DEFAULT 240,
    ADD COLUMN IF NOT EXISTS "exception_due_critical_minutes" integer NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS "exception_escalation_minutes" integer NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS "exception_max_escalations" smallint NOT NULL DEFAULT 3,
    ADD COLUMN IF NOT EXISTS "exception_default_assignee_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "exception_escalation_contact_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "exception_missed_stop_grace_minutes" smallint NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS "exception_pod_grace_hours" smallint NOT NULL DEFAULT 24;

--bun:split
ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "ck_dispatch_controls_exception_due" CHECK (
        "exception_due_info_minutes" BETWEEN 5 AND 10080
        AND "exception_due_warning_minutes" BETWEEN 5 AND 10080
        AND "exception_due_critical_minutes" BETWEEN 5 AND 10080
    ),
    ADD CONSTRAINT "ck_dispatch_controls_exception_escalation" CHECK (
        "exception_escalation_minutes" BETWEEN 5 AND 1440
        AND "exception_max_escalations" BETWEEN 0 AND 10
    ),
    ADD CONSTRAINT "ck_dispatch_controls_exception_sources" CHECK (
        "exception_missed_stop_grace_minutes" BETWEEN 0 AND 1440
        AND "exception_pod_grace_hours" BETWEEN 1 AND 720
    ),
    ADD CONSTRAINT "fk_dispatch_controls_exception_default_assignee" FOREIGN KEY ("exception_default_assignee_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    ADD CONSTRAINT "fk_dispatch_controls_exception_escalation_contact" FOREIGN KEY ("exception_escalation_contact_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
SELECT trenova_rls.reconcile();
