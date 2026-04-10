UPDATE billing_queue_items
SET rerate_variance_percent = 0
WHERE rerate_variance_percent IS NULL;

--bun:split
ALTER TABLE billing_queue_items
    ALTER COLUMN rerate_variance_percent SET DEFAULT 0,
    ALTER COLUMN rerate_variance_percent SET NOT NULL;
