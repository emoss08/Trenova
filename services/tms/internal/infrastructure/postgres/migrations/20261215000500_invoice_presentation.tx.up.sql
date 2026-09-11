--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- How an invoice reads is stamped when it is billed rather than read back from
-- the customer at render time. A customer who changes their presentation
-- preference next month must not silently change how an invoice they were
-- already sent reads.
ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "detail" invoice_detail_enum NOT NULL DEFAULT 'Detailed',
    ADD COLUMN IF NOT EXISTS "section_by" invoice_section_key_enum NOT NULL DEFAULT 'Shipment';
