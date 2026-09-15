-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261221000000_detention_charge_consolidation.tx.up.sql

ALTER TABLE "additional_charges" ADD COLUMN "is_detention" INTEGER NOT NULL DEFAULT 0;

--bun:split

UPDATE
    "additional_charges"
SET
    "is_detention" = TRUE
WHERE
    "detention_occurrence_id" IS NOT NULL;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_additional_charges_id" ON "additional_charges" ("id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_additional_charge" ON "detention_occurrences" ("additional_charge_id")WHERE
    "additional_charge_id" IS NOT NULL;

--bun:split

DROP INDEX IF EXISTS "idx_additional_charges_detention_occurrence";

--bun:split

DROP INDEX IF EXISTS "idx_additional_charges_detention_occurrence";

--bun:split

ALTER TABLE "additional_charges" DROP COLUMN "detention_occurrence_id";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_detention" ON "additional_charges" ("shipment_id", "accessorial_charge_id", "organization_id", "business_unit_id")WHERE
    "is_detention";
