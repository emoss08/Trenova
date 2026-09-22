ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "status_update_preference" VARCHAR(30) NOT NULL DEFAULT 'None';

--bun:split

ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "status_update_recipients" TEXT;

--bun:split

ALTER TABLE "customers"
    DROP CONSTRAINT IF EXISTS "chk_customers_status_update_preference";

--bun:split

ALTER TABLE "customers"
    ADD CONSTRAINT "chk_customers_status_update_preference" CHECK (
        "status_update_preference" IN (
            'None', 'Arrivals', 'Departures', 'ArrivalsAndDepartures'
        )
    );
