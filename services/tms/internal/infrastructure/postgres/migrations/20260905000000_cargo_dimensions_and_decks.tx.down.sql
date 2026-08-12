DROP INDEX IF EXISTS "idx_shipments_envelope_oversize";

--bun:split
ALTER TABLE "shipments"
    DROP CONSTRAINT IF EXISTS "chk_shipments_envelope_positive";

--bun:split
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "envelope_length_feet",
    DROP COLUMN IF EXISTS "envelope_width_feet",
    DROP COLUMN IF EXISTS "envelope_height_feet",
    DROP COLUMN IF EXISTS "envelope_overall_height_feet";

--bun:split
ALTER TABLE "shipment_commodities"
    DROP CONSTRAINT IF EXISTS "chk_shipment_commodities_dimensions_positive";

--bun:split
ALTER TABLE "shipment_commodities"
    DROP COLUMN IF EXISTS "length_feet",
    DROP COLUMN IF EXISTS "width_feet",
    DROP COLUMN IF EXISTS "height_feet";

--bun:split
ALTER TABLE "commodities"
    DROP CONSTRAINT IF EXISTS "chk_commodities_dimensions_positive";

--bun:split
ALTER TABLE "commodities"
    DROP COLUMN IF EXISTS "length_per_unit",
    DROP COLUMN IF EXISTS "width_per_unit",
    DROP COLUMN IF EXISTS "height_per_unit";

--bun:split
ALTER TABLE "equipment_types"
    DROP CONSTRAINT IF EXISTS "chk_equipment_types_deck_height",
    DROP CONSTRAINT IF EXISTS "chk_equipment_types_well_length",
    DROP CONSTRAINT IF EXISTS "chk_equipment_types_well_requires_deck",
    DROP CONSTRAINT IF EXISTS "chk_equipment_types_axle_count";

--bun:split
ALTER TABLE "equipment_types"
    DROP COLUMN IF EXISTS "deck_type",
    DROP COLUMN IF EXISTS "deck_height_inches",
    DROP COLUMN IF EXISTS "well_length_feet",
    DROP COLUMN IF EXISTS "axle_count";

--bun:split
DROP TYPE IF EXISTS "equipment_deck_type_enum";
