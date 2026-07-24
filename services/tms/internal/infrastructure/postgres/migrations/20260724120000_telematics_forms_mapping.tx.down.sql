DROP TABLE IF EXISTS "telematics_form_mapping_items";

DROP TABLE IF EXISTS "telematics_form_mappings";

DROP TABLE IF EXISTS "telematics_form_submissions";

ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "enable_auto_stop_actuals";
