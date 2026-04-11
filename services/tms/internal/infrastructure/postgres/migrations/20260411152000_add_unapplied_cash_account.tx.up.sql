ALTER TABLE accounting_controls
    ADD COLUMN IF NOT EXISTS default_unapplied_cash_account_id VARCHAR(100);
