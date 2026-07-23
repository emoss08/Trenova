ALTER TABLE accounting_controls
    DROP COLUMN IF EXISTS default_driver_reimbursement_account_id;

--bun:split
ALTER TABLE driver_settlement_lines
    DROP COLUMN IF EXISTS recurring_earning_id;

--bun:split
DROP TABLE IF EXISTS recurring_earnings;
