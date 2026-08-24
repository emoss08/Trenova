-- The legacy detention path writes a system charge that no owner column can
-- name: it is generated from the shipment's own stop times, not from a
-- detention occurrence, a fuel program or a contract accessorial. Insisting on
-- exactly one owner outlawed it, which is stricter than the rule the constraint
-- exists to enforce. What must never happen is one charge claimed by two
-- engines, so the count is capped rather than fixed.
ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_single_owner";

--bun:split
ALTER TABLE "additional_charges"
    ADD CONSTRAINT "chk_additional_charges_single_owner" CHECK (NOT "is_system_generated" OR (("fuel_surcharge_program_id" IS NOT NULL)::int + ("detention_occurrence_id" IS NOT NULL)::int + ("rate_agreement_accessorial_id" IS NOT NULL)::int) <= 1);
