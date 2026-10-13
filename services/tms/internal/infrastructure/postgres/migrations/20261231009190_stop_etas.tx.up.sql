--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- The projected arrival at every open stop of an active shipment, worked out
-- from the truck's position, the carrier's check calls or the last departure,
-- with routed drive time and the driver's hours. One row per stop, replaced
-- whenever an input changes and on a periodic sweep, so the board, the
-- shipment page, the API and the agents all read the same answer.
CREATE TABLE IF NOT EXISTS "stop_etas"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "shipment_move_id" varchar(100) NOT NULL,
    "stop_id" varchar(100) NOT NULL,
    "move_sequence" integer NOT NULL,
    "stop_sequence" integer NOT NULL,
    "arrived" boolean NOT NULL DEFAULT false,
    "estimated_arrival" bigint,
    "estimated_departure" bigint,
    "window_end" bigint,
    "slack_minutes" integer,
    "verdict" varchar(20) NOT NULL,
    "basis" varchar(20) NOT NULL,
    "drive_method" varchar(20) NOT NULL,
    "miles_remaining" double precision NOT NULL DEFAULT 0,
    "drive_minutes" integer NOT NULL DEFAULT 0,
    "rest_minutes" integer NOT NULL DEFAULT 0,
    "input_at" bigint,
    "input_stale" boolean NOT NULL DEFAULT false,
    "threshold_minutes" integer NOT NULL,
    "reason" varchar(500),
    "computed_at" bigint NOT NULL,
    CONSTRAINT "pk_stop_etas" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_stop_etas_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_stop_etas_move" FOREIGN KEY ("shipment_move_id", "organization_id", "business_unit_id") REFERENCES "shipment_moves"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_stop_etas_stop" FOREIGN KEY ("stop_id", "organization_id", "business_unit_id") REFERENCES "stops"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_stop_etas_verdict" CHECK ("verdict" IN ('OnTime', 'AtRisk', 'Late', 'Unknown')),
    CONSTRAINT "chk_stop_etas_basis" CHECK ("basis" IN ('Position', 'CheckCall', 'CarrierReported', 'AtStop', 'LastDeparture', 'None')),
    CONSTRAINT "chk_stop_etas_drive_method" CHECK ("drive_method" IN ('Routed', 'Planning', 'Reported', 'None')),
    CONSTRAINT "chk_stop_etas_amounts" CHECK ("miles_remaining" >= 0 AND "drive_minutes" >= 0 AND "rest_minutes" >= 0),
    CONSTRAINT "chk_stop_etas_threshold" CHECK ("threshold_minutes" BETWEEN 0 AND 1440),
    CONSTRAINT "chk_stop_etas_computed_at" CHECK ("computed_at" > 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_stop_etas_stop"
    ON "stop_etas"("organization_id", "business_unit_id", "stop_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_stop_etas_shipment"
    ON "stop_etas"("organization_id", "business_unit_id", "shipment_id", "move_sequence", "stop_sequence");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_stop_etas_attention"
    ON "stop_etas"("organization_id", "business_unit_id", "shipment_move_id")
    WHERE "verdict" IN ('AtRisk', 'Late') AND NOT "arrived";

--bun:split
-- How much slack under a stop's window counts as at risk, how long a stop
-- takes once the truck is there, and whether an at-risk load is raised in the
-- watchtower. A customer can ask for a tighter or looser threshold.
ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "enable_eta_alerts" boolean NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS "eta_risk_threshold_minutes" smallint NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS "eta_dwell_minutes" smallint NOT NULL DEFAULT 60;

--bun:split
ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "chk_dispatch_controls_eta_risk_threshold_minutes" CHECK ("eta_risk_threshold_minutes" BETWEEN 0 AND 1440),
    ADD CONSTRAINT "chk_dispatch_controls_eta_dwell_minutes" CHECK ("eta_dwell_minutes" BETWEEN 0 AND 720);

--bun:split
ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "eta_risk_threshold_minutes" smallint;

--bun:split
ALTER TABLE "customers"
    ADD CONSTRAINT "chk_customers_eta_risk_threshold_minutes" CHECK ("eta_risk_threshold_minutes" IS NULL OR "eta_risk_threshold_minutes" BETWEEN 0 AND 1440);

--bun:split
SELECT trenova_rls.reconcile();
