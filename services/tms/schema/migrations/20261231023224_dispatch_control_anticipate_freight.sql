-- Whether plans position drivers for freight the organization's forecast says is coming.
-- Off by default: it uses shipment history in a new way, and it is honoured only when the
-- forecast has proven itself (demand_forecast_quality.status = 'ready').
ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "anticipate_freight" boolean NOT NULL DEFAULT false;
