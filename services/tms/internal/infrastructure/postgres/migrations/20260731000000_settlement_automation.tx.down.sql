DROP INDEX IF EXISTS idx_driver_pay_events_on_hold;

ALTER TABLE driver_pay_events
    DROP COLUMN IF EXISTS on_hold,
    DROP COLUMN IF EXISTS hold_reason;

ALTER TABLE settlement_controls
    DROP COLUMN IF EXISTS auto_attach_accruals,
    DROP COLUMN IF EXISTS auto_post_on_approve;
