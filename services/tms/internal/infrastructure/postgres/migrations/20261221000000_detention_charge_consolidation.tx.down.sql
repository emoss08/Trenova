DROP INDEX IF EXISTS "idx_additional_charges_detention";

--bun:split
ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_single_owner";

--bun:split
ALTER TABLE "additional_charges"
    ADD COLUMN IF NOT EXISTS "detention_occurrence_id" varchar(100);

--bun:split
-- A charge that bills exactly one occurrence gets its link back; one that
-- folded several stops has no single occurrence to name.
UPDATE
    "additional_charges" ac
SET
    "detention_occurrence_id" = o."occurrence_id"
FROM (
    SELECT
        "additional_charge_id",
        "organization_id",
        "business_unit_id",
        MIN("id") AS "occurrence_id",
        COUNT(*) AS "cnt"
    FROM
        "detention_occurrences"
    WHERE
        "additional_charge_id" IS NOT NULL
    GROUP BY
        "additional_charge_id",
        "organization_id",
        "business_unit_id") o
WHERE
    o."cnt" = 1
    AND o."additional_charge_id" = ac."id"
    AND o."organization_id" = ac."organization_id"
    AND o."business_unit_id" = ac."business_unit_id";

--bun:split
ALTER TABLE "additional_charges"
    ADD CONSTRAINT "fk_additional_charges_detention_occurrence" FOREIGN KEY ("detention_occurrence_id", "organization_id", "business_unit_id") REFERENCES "detention_occurrences"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_additional_charges_detention_occurrence" ON "additional_charges"("detention_occurrence_id")
WHERE
    "detention_occurrence_id" IS NOT NULL;

--bun:split
ALTER TABLE "additional_charges"
    ADD CONSTRAINT "chk_additional_charges_single_owner" CHECK (NOT "is_system_generated" OR (("fuel_surcharge_program_id" IS NOT NULL)::int + ("detention_occurrence_id" IS NOT NULL)::int + ("rate_agreement_accessorial_id" IS NOT NULL)::int) <= 1);

--bun:split
ALTER TABLE "additional_charges"
    DROP COLUMN IF EXISTS "is_detention";

--bun:split
ALTER TABLE "detention_occurrences"
    DROP CONSTRAINT IF EXISTS "fk_detention_occurrences_additional_charge";

--bun:split
DROP INDEX IF EXISTS "idx_detention_occurrences_additional_charge";

--bun:split
DROP INDEX IF EXISTS "uq_additional_charges_id";
