--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE billing_controls ADD COLUMN IF NOT EXISTS invoice_number_prefix VARCHAR(10) NOT NULL DEFAULT 'INV-';

--bun:split
ALTER TABLE billing_controls ADD COLUMN IF NOT EXISTS credit_memo_number_prefix VARCHAR(10) NOT NULL DEFAULT 'CM-';
