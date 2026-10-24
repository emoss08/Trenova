ALTER TABLE driver_settlements
    ADD COLUMN IF NOT EXISTS paid_journal_batch_id VARCHAR(100);
