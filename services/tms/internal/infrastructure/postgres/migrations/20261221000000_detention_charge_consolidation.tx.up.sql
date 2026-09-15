-- A shipment carries one detention charge per accessorial, however many stops
-- were detained. The charge is marked rather than pointed at a single
-- occurrence, and every occurrence it bills points back at it.
ALTER TABLE "additional_charges"
    ADD COLUMN IF NOT EXISTS "is_detention" boolean NOT NULL DEFAULT FALSE;

--bun:split
UPDATE
    "additional_charges"
SET
    "is_detention" = TRUE
WHERE
    "detention_occurrence_id" IS NOT NULL;

--bun:split
-- The legacy path wrote its detention charge with no owner column at all.
UPDATE
    "additional_charges" ac
SET
    "is_detention" = TRUE
FROM
    "shipment_controls" sc
WHERE
    sc."organization_id" = ac."organization_id"
    AND sc."business_unit_id" = ac."business_unit_id"
    AND sc."detention_charge_id" = ac."accessorial_charge_id"
    AND ac."is_system_generated"
    AND NOT ac."is_detention"
    AND ac."fuel_surcharge_program_id" IS NULL
    AND ac."rate_agreement_accessorial_id" IS NULL;

--bun:split
UPDATE
    "detention_occurrences" o
SET
    "additional_charge_id" = ac."id"
FROM
    "additional_charges" ac
WHERE
    ac."detention_occurrence_id" = o."id"
    AND ac."organization_id" = o."organization_id"
    AND ac."business_unit_id" = o."business_unit_id";

--bun:split
-- Shipments that already carry several detention charges for one accessorial
-- fold into the oldest row: the occurrences move to it, its amount becomes the
-- group total, and the later rows go.
WITH "ranked" AS (
    SELECT
        ac."id",
        ac."organization_id",
        ac."business_unit_id",
        FIRST_VALUE(ac."id") OVER "grp" AS "keeper_id",
        COUNT(*) OVER "grp" AS "cnt"
    FROM
        "additional_charges" ac
    WHERE
        ac."is_detention"
    WINDOW "grp" AS (PARTITION BY ac."shipment_id", ac."organization_id", ac."business_unit_id", ac."accessorial_charge_id" ORDER BY ac."created_at", ac."id" ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING))
UPDATE
    "detention_occurrences" o
SET
    "additional_charge_id" = r."keeper_id"
FROM
    "ranked" r
WHERE
    r."cnt" > 1
    AND r."id" <> r."keeper_id"
    AND o."additional_charge_id" = r."id"
    AND o."organization_id" = r."organization_id"
    AND o."business_unit_id" = r."business_unit_id";

--bun:split
WITH "ranked" AS (
    SELECT
        ac."id",
        ac."organization_id",
        ac."business_unit_id",
        FIRST_VALUE(ac."id") OVER "grp" AS "keeper_id",
        COUNT(*) OVER "grp" AS "cnt",
        SUM(ac."amount" * GREATEST(ac."unit", 1)) OVER "grp" AS "group_total"
    FROM
        "additional_charges" ac
    WHERE
        ac."is_detention"
    WINDOW "grp" AS (PARTITION BY ac."shipment_id", ac."organization_id", ac."business_unit_id", ac."accessorial_charge_id" ORDER BY ac."created_at", ac."id" ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING))
UPDATE
    "additional_charges" ac
SET
    "method" = 'Flat',
    "unit" = 1,
    "amount" = r."group_total",
    "updated_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
FROM
    "ranked" r
WHERE
    r."cnt" > 1
    AND r."id" = r."keeper_id"
    AND ac."id" = r."id"
    AND ac."organization_id" = r."organization_id"
    AND ac."business_unit_id" = r."business_unit_id";

--bun:split
WITH "ranked" AS (
    SELECT
        ac."id",
        ac."organization_id",
        ac."business_unit_id",
        FIRST_VALUE(ac."id") OVER "grp" AS "keeper_id",
        COUNT(*) OVER "grp" AS "cnt"
    FROM
        "additional_charges" ac
    WHERE
        ac."is_detention"
    WINDOW "grp" AS (PARTITION BY ac."shipment_id", ac."organization_id", ac."business_unit_id", ac."accessorial_charge_id" ORDER BY ac."created_at", ac."id" ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING))
DELETE FROM "additional_charges" ac USING "ranked" r
WHERE r."cnt" > 1
    AND r."id" <> r."keeper_id"
    AND ac."id" = r."id"
    AND ac."organization_id" = r."organization_id"
    AND ac."business_unit_id" = r."business_unit_id";

--bun:split
UPDATE
    "detention_occurrences" o
SET
    "additional_charge_id" = NULL
WHERE
    o."additional_charge_id" IS NOT NULL
    AND NOT EXISTS (
        SELECT
            1
        FROM
            "additional_charges" ac
        WHERE
            ac."id" = o."additional_charge_id");

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_additional_charges_id" ON "additional_charges"("id");

--bun:split
ALTER TABLE "detention_occurrences"
    DROP CONSTRAINT IF EXISTS "fk_detention_occurrences_additional_charge";

--bun:split
ALTER TABLE "detention_occurrences"
    ADD CONSTRAINT "fk_detention_occurrences_additional_charge" FOREIGN KEY ("additional_charge_id") REFERENCES "additional_charges"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_detention_occurrences_additional_charge" ON "detention_occurrences"("additional_charge_id")
WHERE
    "additional_charge_id" IS NOT NULL;

--bun:split
ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_single_owner";

--bun:split
ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "fk_additional_charges_detention_occurrence";

--bun:split
DROP INDEX IF EXISTS "idx_additional_charges_detention_occurrence";

--bun:split
ALTER TABLE "additional_charges"
    DROP COLUMN IF EXISTS "detention_occurrence_id";

--bun:split
ALTER TABLE "additional_charges"
    ADD CONSTRAINT "chk_additional_charges_single_owner" CHECK (NOT "is_system_generated" OR (("fuel_surcharge_program_id" IS NOT NULL)::int + "is_detention"::int + ("rate_agreement_accessorial_id" IS NOT NULL)::int) <= 1);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_additional_charges_detention" ON "additional_charges"("shipment_id", "accessorial_charge_id", "organization_id", "business_unit_id")
WHERE
    "is_detention";
