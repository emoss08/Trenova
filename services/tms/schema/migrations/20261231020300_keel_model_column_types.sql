-- The keel models read these columns with the Go types below; keel.Verify holds each column to its field.
ALTER TABLE "additional_charges"
    ALTER COLUMN "unit" TYPE smallint;

ALTER TABLE "dash_controls"
    ALTER COLUMN "detention_alert_threshold_minutes" TYPE smallint;

ALTER TABLE "shipment_controls"
    ALTER COLUMN "auto_delay_shipments_threshold" TYPE smallint,
    ALTER COLUMN "detention_threshold" TYPE smallint;

-- Calendar dates are ISO text, as briefing dates are: a rate or price belongs to the day it is quoted for.
ALTER TABLE "exchange_rates"
    ALTER COLUMN "date" TYPE varchar(10) USING to_char("date", 'YYYY-MM-DD'),
    ADD CONSTRAINT "ck_exchange_rates_date" CHECK ("date" ~ '^\d{4}-\d{2}-\d{2}$');

ALTER TABLE "fuel_index_prices"
    ALTER COLUMN "price_date" TYPE varchar(10) USING to_char("price_date", 'YYYY-MM-DD'),
    ADD CONSTRAINT "ck_fuel_index_prices_price_date" CHECK ("price_date" ~ '^\d{4}-\d{2}-\d{2}$');

-- Both generated columns always produce a value; saying so lets a non-pointer field read them.
ALTER TABLE "invoices"
    ALTER COLUMN "balance_due_minor" SET NOT NULL;

ALTER TABLE "shipments"
    ALTER COLUMN "stage_rank" SET NOT NULL;

ALTER TABLE "workers"
    ALTER COLUMN "whole_name" SET NOT NULL;

UPDATE "telematics_form_mappings" SET "version" = 0 WHERE "version" IS NULL;

ALTER TABLE "telematics_form_mappings"
    ALTER COLUMN "version" SET DEFAULT 0,
    ALTER COLUMN "version" SET NOT NULL;
