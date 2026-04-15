ALTER TABLE bank_receipts DROP CONSTRAINT IF EXISTS fk_bank_receipts_import_batch;

--bun:split
ALTER TABLE bank_receipts DROP COLUMN IF EXISTS import_batch_id;

--bun:split
DROP TABLE IF EXISTS bank_receipt_import_batches;
