-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260724000000_fuel_surcharge_percent_basis_customer_mode.tx.up.sql

ALTER TABLE "fuel_surcharge_programs" ADD COLUMN "percent_basis" TEXT NOT NULL DEFAULT 'Linehaul';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "fuel_surcharge_mode" TEXT NOT NULL DEFAULT 'None';

--bun:split

UPDATE
    "customer_billing_profiles"
SET
    "fuel_surcharge_mode" = 'Program'
WHERE
    "fuel_surcharge_program_id" IS NOT NULL;
