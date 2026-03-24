-- Add composite index for GetExpiredOpenPeriods query optimization
-- This index covers the exact columns used in the WHERE clause of the query
CREATE INDEX IF NOT EXISTS idx_fiscal_periods_expired_open
    ON "fiscal_periods"("organization_id", "business_unit_id", "status", "end_date")
    WHERE "status" = 'Open';

COMMENT ON INDEX idx_fiscal_periods_expired_open IS 'Optimizes queries for finding expired open fiscal periods in auto-close workflows';
