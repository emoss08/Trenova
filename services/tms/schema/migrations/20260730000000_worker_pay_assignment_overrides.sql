ALTER TABLE worker_pay_assignments
    ADD COLUMN IF NOT EXISTS rate_overrides JSONB;
