DROP TABLE IF EXISTS driver_expenses;

--bun:split
ALTER TABLE assignments
    DROP CONSTRAINT IF EXISTS chk_assignments_ack_status;

--bun:split
ALTER TABLE assignments
    DROP COLUMN IF EXISTS ack_status,
    DROP COLUMN IF EXISTS ack_at,
    DROP COLUMN IF EXISTS ack_reason;
