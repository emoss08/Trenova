--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A check call is one report of where a load is and how it is doing: a
-- dispatcher's note of a phone call, a carrier posting through its tracking
-- link, or an agent. An arrival or departure also becomes a tracking event,
-- linked here, and settles the stop's times with every other source.
CREATE TABLE IF NOT EXISTS "check_calls"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "stop_id" varchar(100),
    "source" varchar(20) NOT NULL,
    "source_key" varchar(255) NOT NULL,
    "kind" varchar(20) NOT NULL,
    "reported_at" bigint NOT NULL,
    "received_at" bigint NOT NULL,
    "latitude" double precision,
    "longitude" double precision,
    "city" varchar(100),
    "state_code" varchar(10),
    "delayed" boolean NOT NULL DEFAULT false,
    "delay_reason" varchar(255),
    "estimated_arrival" bigint,
    "note" text,
    "contact_name" varchar(100),
    "reported_by_id" varchar(100),
    "tracking_link_id" varchar(100),
    "tracking_event_id" varchar(100),
    "created_at" bigint NOT NULL,
    CONSTRAINT "pk_check_calls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_check_calls_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_check_calls_source" CHECK ("source" IN ('Dispatcher', 'Carrier', 'Agent')),
    CONSTRAINT "chk_check_calls_kind" CHECK (
        "kind" IN ('Update', 'Arrival', 'Departure')
        AND ("kind" = 'Update' OR "stop_id" IS NOT NULL)
    ),
    CONSTRAINT "chk_check_calls_times" CHECK ("reported_at" > 0 AND "received_at" > 0),
    CONSTRAINT "chk_check_calls_position" CHECK (
        ("latitude" IS NULL AND "longitude" IS NULL)
        OR (
            "latitude" BETWEEN -90 AND 90
            AND "longitude" BETWEEN -180 AND 180
        )
    ),
    CONSTRAINT "chk_check_calls_delay" CHECK (NOT "delayed" OR btrim(coalesce("delay_reason", '')) <> ''),
    CONSTRAINT "chk_check_calls_source_key" CHECK (btrim("source_key") <> '')
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_check_calls_source_key"
    ON "check_calls"("organization_id", "business_unit_id", "source", "source_key");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_check_calls_move_reported"
    ON "check_calls"("organization_id", "business_unit_id", "shipment_move_id", "received_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_check_calls_shipment"
    ON "check_calls"("organization_id", "business_unit_id", "shipment_id", "reported_at" DESC);

--bun:split
-- A carrier's tracking link: one live link per move, opened without signing in.
-- Only the token's hash is kept; issuing a new link revokes the last one.
CREATE TABLE IF NOT EXISTS "carrier_tracking_links"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "carrier_id" varchar(100) NOT NULL,
    "token_hash" varchar(128) NOT NULL,
    "recipient_email" varchar(320),
    "expires_at" bigint NOT NULL,
    "revoked_at" bigint,
    "last_used_at" bigint,
    "created_by_id" varchar(100),
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_carrier_tracking_links" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_carrier_tracking_links_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_carrier_tracking_links_expiry" CHECK ("expires_at" > "created_at")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_tracking_links_hash"
    ON "carrier_tracking_links"("token_hash");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_tracking_links_move"
    ON "carrier_tracking_links"("organization_id", "business_unit_id", "shipment_move_id", "created_at" DESC);

--bun:split
COMMENT ON TABLE "carrier_tracking_links" IS 'Hashed carrier tracking link tokens. The raw token only ever appears in the link sent to the carrier.';

--bun:split
ALTER TABLE "tracking_events"
    DROP CONSTRAINT IF EXISTS "chk_tracking_events_source";

--bun:split
ALTER TABLE "tracking_events"
    ADD CONSTRAINT "chk_tracking_events_source" CHECK ("source" IN ('Dispatcher', 'Agent', 'Driver', 'Carrier', 'Telematics', 'EDI'));

--bun:split
-- How long a brokered load in transit may go without a check call before it
-- is raised for a dispatcher. A customer can ask for a tighter or looser one.
ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "enable_check_call_monitoring" boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS "check_call_interval_hours" smallint NOT NULL DEFAULT 4;

--bun:split
ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "chk_dispatch_controls_check_call_interval_hours" CHECK ("check_call_interval_hours" BETWEEN 1 AND 72);

--bun:split
ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "check_call_interval_hours" smallint;

--bun:split
ALTER TABLE "customers"
    ADD CONSTRAINT "chk_customers_check_call_interval_hours" CHECK ("check_call_interval_hours" IS NULL OR "check_call_interval_hours" BETWEEN 1 AND 72);

--bun:split
SELECT trenova_rls.reconcile();
