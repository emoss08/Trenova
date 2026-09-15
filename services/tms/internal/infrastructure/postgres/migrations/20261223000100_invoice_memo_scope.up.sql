-- A standalone memo bills a customer with no shipment behind it. Adding an enum
-- value cannot run inside a transaction, so this file is non-transactional.
ALTER TYPE "invoice_scope_enum" ADD VALUE IF NOT EXISTS 'Memo';
