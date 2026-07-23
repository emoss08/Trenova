DROP TABLE IF EXISTS driver_pay_events;

--bun:split
DROP TABLE IF EXISTS driver_settlement_lines;

--bun:split
DROP TABLE IF EXISTS driver_settlements;

--bun:split
DROP TABLE IF EXISTS driver_settlement_batches;

--bun:split
DROP TABLE IF EXISTS pay_advances;

--bun:split
DROP TABLE IF EXISTS recurring_deductions;

--bun:split
DROP TABLE IF EXISTS escrow_transactions;

--bun:split
DROP TABLE IF EXISTS escrow_accounts;

--bun:split
DROP TABLE IF EXISTS worker_pay_assignments;

--bun:split
DROP TABLE IF EXISTS driver_pay_profile_components;

--bun:split
DROP TABLE IF EXISTS driver_pay_profiles;

--bun:split
DROP TABLE IF EXISTS settlement_controls;

--bun:split
ALTER TABLE tractors
    DROP CONSTRAINT IF EXISTS fk_tractors_owner_worker,
    DROP COLUMN IF EXISTS ownership_type,
    DROP COLUMN IF EXISTS owner_worker_id,
    DROP COLUMN IF EXISTS lessor_name,
    DROP COLUMN IF EXISTS lease_reference,
    DROP COLUMN IF EXISTS lease_end_date;

--bun:split
ALTER TABLE trailers
    DROP CONSTRAINT IF EXISTS fk_trailers_owner_worker,
    DROP COLUMN IF EXISTS ownership_type,
    DROP COLUMN IF EXISTS owner_worker_id,
    DROP COLUMN IF EXISTS lessor_name,
    DROP COLUMN IF EXISTS lease_reference,
    DROP COLUMN IF EXISTS lease_end_date;

--bun:split
ALTER TABLE accounting_controls
    DROP COLUMN IF EXISTS default_driver_pay_expense_account_id,
    DROP COLUMN IF EXISTS default_purchased_transportation_account_id,
    DROP COLUMN IF EXISTS default_settlements_payable_account_id,
    DROP COLUMN IF EXISTS default_driver_advance_account_id,
    DROP COLUMN IF EXISTS default_escrow_liability_account_id,
    DROP COLUMN IF EXISTS default_escrow_interest_expense_account_id;
