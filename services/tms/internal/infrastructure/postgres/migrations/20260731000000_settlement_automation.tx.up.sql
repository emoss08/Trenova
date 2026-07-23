ALTER TABLE driver_pay_events
    ADD COLUMN IF NOT EXISTS on_hold BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS hold_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_driver_pay_events_on_hold
    ON driver_pay_events (organization_id, business_unit_id, worker_id)
    WHERE on_hold = true;

ALTER TABLE settlement_controls
    ADD COLUMN IF NOT EXISTS auto_attach_accruals BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS auto_post_on_approve BOOLEAN NOT NULL DEFAULT false;
