ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_single_owner";

--bun:split
ALTER TABLE "additional_charges"
    ADD CONSTRAINT "chk_additional_charges_single_owner" CHECK (NOT "is_system_generated" OR (("fuel_surcharge_program_id" IS NOT NULL)::int + ("detention_occurrence_id" IS NOT NULL)::int + ("rate_agreement_accessorial_id" IS NOT NULL)::int) = 1);
