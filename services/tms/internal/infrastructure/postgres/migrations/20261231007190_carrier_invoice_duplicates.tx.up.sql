--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A carrier invoice is identified by who sent it and its number, compared without
-- regard to case or stray spacing. A partner that resends a 210, or an inbound file
-- reprocessed after a warning, used to record the same invoice a second time; rows
-- already recorded twice are kept, pointed at the first copy, and left out of the
-- uniqueness that new rows now obey.
ALTER TABLE "edi_carrier_invoices"
    ADD COLUMN IF NOT EXISTS "invoice_number_key" varchar(100),
    ADD COLUMN IF NOT EXISTS "duplicate_of_id" varchar(100);

--bun:split
UPDATE
    "edi_carrier_invoices"
SET
    "invoice_number_key" = upper(regexp_replace(btrim("invoice_number"), '\s+', ' ', 'g'));

--bun:split
WITH ranked AS (
    SELECT
        "id",
        "organization_id",
        "business_unit_id",
        first_value("id") OVER (PARTITION BY "organization_id", "business_unit_id", "edi_partner_id", "invoice_number_key" ORDER BY "created_at", "id") AS "first_id"
    FROM
        "edi_carrier_invoices")
UPDATE
    "edi_carrier_invoices" AS eci
SET
    "duplicate_of_id" = ranked."first_id"
FROM
    ranked
WHERE
    eci."id" = ranked."id"
    AND eci."organization_id" = ranked."organization_id"
    AND eci."business_unit_id" = ranked."business_unit_id"
    AND ranked."first_id" <> ranked."id";

--bun:split
ALTER TABLE "edi_carrier_invoices"
    ALTER COLUMN "invoice_number_key" SET NOT NULL;

--bun:split
DROP INDEX IF EXISTS "idx_edi_carrier_invoices_partner_invoice_number";

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_edi_carrier_invoices_partner_invoice_number" ON "edi_carrier_invoices"("organization_id", "business_unit_id", "edi_partner_id", "invoice_number_key")
WHERE
    "duplicate_of_id" IS NULL;

--bun:split
-- A match pays a carrier invoice, so one invoice number may hold only one live match
-- per carrier, whichever source it came from. Live matches that already share a
-- number are pointed at the first and cannot be accepted while it stands. A second
-- invoice matched to a load that already has one is flagged as a possible duplicate
-- for a person to judge; it is never accepted automatically.
ALTER TABLE "carrier_invoice_matches"
    ADD COLUMN IF NOT EXISTS "invoice_number_key" varchar(100),
    ADD COLUMN IF NOT EXISTS "duplicate_of_match_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "possible_duplicate_of_id" varchar(100);

--bun:split
UPDATE
    "carrier_invoice_matches"
SET
    "invoice_number_key" = nullif(upper(regexp_replace(btrim("invoice_number"), '\s+', ' ', 'g')), '');

--bun:split
WITH ranked AS (
    SELECT
        "id",
        "organization_id",
        "business_unit_id",
        first_value("id") OVER (PARTITION BY "organization_id", "business_unit_id", "carrier_id", "invoice_number_key" ORDER BY "created_at", "id") AS "first_id"
    FROM
        "carrier_invoice_matches"
    WHERE
        "invoice_number_key" IS NOT NULL
        AND "status" <> 'Rejected')
UPDATE
    "carrier_invoice_matches" AS cim
SET
    "duplicate_of_match_id" = ranked."first_id"
FROM
    ranked
WHERE
    cim."id" = ranked."id"
    AND cim."organization_id" = ranked."organization_id"
    AND cim."business_unit_id" = ranked."business_unit_id"
    AND ranked."first_id" <> ranked."id";

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_carrier_invoice_matches_carrier_invoice_number" ON "carrier_invoice_matches"("organization_id", "business_unit_id", "carrier_id", "invoice_number_key")
WHERE
    "invoice_number_key" IS NOT NULL
    AND "status" <> 'Rejected'
    AND "duplicate_of_match_id" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_carrier_invoice_matches_assignment" ON "carrier_invoice_matches"("organization_id", "business_unit_id", "carrier_assignment_id")
WHERE
    "status" <> 'Rejected';
