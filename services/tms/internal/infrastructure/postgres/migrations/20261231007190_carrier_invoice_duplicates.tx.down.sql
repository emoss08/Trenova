--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_carrier_invoice_matches_assignment";

--bun:split
DROP INDEX IF EXISTS "uq_carrier_invoice_matches_carrier_invoice_number";

--bun:split
ALTER TABLE "carrier_invoice_matches"
    DROP COLUMN IF EXISTS "possible_duplicate_of_id",
    DROP COLUMN IF EXISTS "duplicate_of_match_id",
    DROP COLUMN IF EXISTS "invoice_number_key";

--bun:split
DROP INDEX IF EXISTS "uq_edi_carrier_invoices_partner_invoice_number";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_edi_carrier_invoices_partner_invoice_number" ON "edi_carrier_invoices"("organization_id", "business_unit_id", "edi_partner_id", "invoice_number");

--bun:split
ALTER TABLE "edi_carrier_invoices"
    DROP COLUMN IF EXISTS "duplicate_of_id",
    DROP COLUMN IF EXISTS "invoice_number_key";
