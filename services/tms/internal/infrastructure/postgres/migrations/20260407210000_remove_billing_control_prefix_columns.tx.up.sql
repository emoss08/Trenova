--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE billing_controls DROP COLUMN IF EXISTS invoice_number_prefix;

--bun:split
ALTER TABLE billing_controls DROP COLUMN IF EXISTS credit_memo_number_prefix;
