ALTER TABLE billing_queue_items
    ALTER COLUMN rerate_variance_percent DROP NOT NULL,
    ALTER COLUMN rerate_variance_percent DROP DEFAULT;

--bun:split
UPDATE billing_queue_items
SET rerate_variance_percent = NULL
WHERE rerate_variance_percent = 0;
