--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- One row per arrival or departure reported for a stop, from any source: a
-- dispatcher, an agent, the driver app, telematics or an EDI 214. A source's own
-- key makes a resent report recognizable, and the stop's actual times are derived
-- from these rows in event-time order, so the order they arrive in does not
-- change the result. The outcome records what the latest derivation made of each
-- row and why.
CREATE TABLE IF NOT EXISTS "tracking_events"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "stop_id" varchar(100) NOT NULL,
    "source" varchar(20) NOT NULL,
    "source_key" varchar(255) NOT NULL,
    "kind" varchar(20) NOT NULL,
    "match_method" varchar(20) NOT NULL,
    "event_at" bigint NOT NULL,
    "received_at" bigint NOT NULL,
    "latitude" double precision,
    "longitude" double precision,
    "source_status" varchar(64),
    "raw_reference" varchar(255),
    "reported_by_id" varchar(100),
    "outcome" varchar(20) NOT NULL,
    "outcome_reason" text,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_tracking_events" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_tracking_events_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_tracking_events_source" CHECK ("source" IN ('Dispatcher', 'Agent', 'Driver', 'Telematics', 'EDI')),
    CONSTRAINT "chk_tracking_events_kind" CHECK ("kind" IN ('Arrival', 'Departure')),
    CONSTRAINT "chk_tracking_events_match_method" CHECK ("match_method" IN ('Direct', 'StopReference', 'Location', 'StopRole', 'Inferred')),
    CONSTRAINT "chk_tracking_events_outcome" CHECK (
        "outcome" IN ('Applied', 'Duplicate', 'Pending', 'Superseded', 'Refused')
        AND (
            "outcome" NOT IN ('Pending', 'Superseded', 'Refused')
            OR btrim(coalesce("outcome_reason", '')) <> ''
        )
    ),
    CONSTRAINT "chk_tracking_events_times" CHECK ("event_at" > 0 AND "received_at" > 0),
    CONSTRAINT "chk_tracking_events_position" CHECK (
        ("latitude" IS NULL AND "longitude" IS NULL)
        OR (
            "latitude" BETWEEN -90 AND 90
            AND "longitude" BETWEEN -180 AND 180
        )
    ),
    CONSTRAINT "chk_tracking_events_source_key" CHECK (btrim("source_key") <> '')
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_tracking_events_source_key"
    ON "tracking_events"("organization_id", "business_unit_id", "source", "source_key");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_tracking_events_move"
    ON "tracking_events"("organization_id", "business_unit_id", "shipment_move_id", "event_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_tracking_events_shipment"
    ON "tracking_events"("organization_id", "business_unit_id", "shipment_id", "event_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_tracking_events_pending"
    ON "tracking_events"("organization_id", "business_unit_id", "received_at")
    WHERE "outcome" = 'Pending';

--bun:split
SELECT trenova_rls.reconcile();
